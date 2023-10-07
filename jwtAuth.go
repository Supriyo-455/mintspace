package main

import (
	"fmt"
	"net/http"
	"time"

	jwt "github.com/golang-jwt/jwt/v4"
	"github.com/julienschmidt/httprouter"
)

// TODO: remove secret from here
var secret = "MY_SECRET"

func createJWT(userLoginRequest *UserLoginRequest) (string, error) {
	claims := &jwt.MapClaims{
		"expiresAt": time.Now().Add(time.Hour * 24 * 30).Unix(),
		"email":     userLoginRequest.Email,
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	return token.SignedString([]byte(secret))
}

func validateJWT(tokenString string) (*jwt.Token, error) {
	return jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		// Don't forget to validate the alg is what you expect:
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}

		// hmacSampleSecret is a []byte containing your secret, e.g. []byte("my_secret_key")
		return []byte(secret), nil
	})
}

func withJWTAuth(handle httprouter.Handle) httprouter.Handle {
	return func(res http.ResponseWriter, req *http.Request, params httprouter.Params) {
		LogInfo().Println("Calling jwt middleware..")

		cookie, err := req.Cookie("Auth")
		if err != nil {
			// redirect to the login page
			http.Redirect(res, req, "/login", http.StatusSeeOther)

			if err == http.ErrNoCookie {
				// If the cookie is not set, return an unauthorized status
				res.WriteHeader(http.StatusUnauthorized)
			}
			// For any other type of error, return a bad request status
			res.WriteHeader(http.StatusBadRequest)
		}

		token, err := validateJWT(cookie.Value)
		if err != nil || !token.Valid {
			// redirect to the login page
			http.Redirect(res, req, "/login", http.StatusSeeOther)
		}
	}
}
