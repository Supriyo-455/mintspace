package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"time"

	jwt "github.com/golang-jwt/jwt/v4"
	"github.com/julienschmidt/httprouter"
)

type contextKey string

const userContextKey contextKey = "sessionUser"

func getJWTSecret() []byte {
	s := os.Getenv("SECRET")
	if s == "" {
		s = os.Getenv("secret")
	}
	if s == "" {
		s = "mintspace-super-secure-research-secret-2026"
	}
	return []byte(s)
}

func createJWT(email, name string, isAdmin bool) (string, error) {
	claims := &jwt.MapClaims{
		"email":     email,
		"name":      name,
		"isAdmin":   isAdmin,
		"expiresAt": time.Now().Add(time.Hour * 24 * 30).Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(getJWTSecret())
}

func validateJWT(tokenString string) (*jwt.Token, *jwt.MapClaims, error) {
	claims := &jwt.MapClaims{}
	token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return getJWTSecret(), nil
	})
	if err != nil {
		return nil, nil, err
	}
	return token, claims, nil
}

// getSessionUser returns the authenticated user from the cookie if present and valid
func getSessionUser(req *http.Request) *SessionUser {
	cookie, err := req.Cookie("Auth")
	if err != nil || cookie.Value == "" {
		return nil
	}

	token, claims, err := validateJWT(cookie.Value)
	if err != nil || !token.Valid {
		return nil
	}

	email, _ := (*claims)["email"].(string)
	name, _ := (*claims)["name"].(string)
	isAdmin, _ := (*claims)["isAdmin"].(bool)

	if email == "" {
		return nil
	}

	return &SessionUser{
		Email:   email,
		Name:    name,
		IsAdmin: isAdmin,
	}
}

// withAuthMiddleware enforces authentication and passes the session user
func withAuthMiddleware(handle httprouter.Handle) httprouter.Handle {
	return func(res http.ResponseWriter, req *http.Request, params httprouter.Params) {
		session := getSessionUser(req)
		if session == nil {
			http.Redirect(res, req, "/login", http.StatusSeeOther)
			return
		}

		ctx := context.WithValue(req.Context(), userContextKey, session)
		handle(res, req.WithContext(ctx), params)
	}
}

// withAdminMiddleware enforces that the user is logged in AND is an admin
func withAdminMiddleware(handle httprouter.Handle) httprouter.Handle {
	return func(res http.ResponseWriter, req *http.Request, params httprouter.Params) {
		session := getSessionUser(req)
		if session == nil {
			http.Redirect(res, req, "/login", http.StatusSeeOther)
			return
		}

		if !session.IsAdmin {
			http.Error(res, "Access Denied: Administrator privileges required.", http.StatusForbidden)
			return
		}

		ctx := context.WithValue(req.Context(), userContextKey, session)
		handle(res, req.WithContext(ctx), params)
	}
}
