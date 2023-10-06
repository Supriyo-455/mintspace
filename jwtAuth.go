package main

import (
	"fmt"
	"net/http"

	jwt "github.com/golang-jwt/jwt/v4"
	"github.com/julienschmidt/httprouter"
)

// TODO: remove secret from here
var secret = "MY_SECRET"

func createJWT(userSignupRequest *UserSignupRequest) (string, error) {
	claims := &jwt.MapClaims{
		"expiresAt": 15000,
		"email":     userSignupRequest.Email,
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	return token.SignedString([]byte(secret))
}

func validateJWT(tokenString string) (*jwt.Token, error) {
	return jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		// Don't forget to validate the alg is what you expect:
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("Unexpected signing method: %v", token.Header["alg"])
		}

		// hmacSampleSecret is a []byte containing your secret, e.g. []byte("my_secret_key")
		return []byte(secret), nil
	})
}

func withJWTAuth(handle httprouter.Handle, s *Storage) httprouter.Handle {
	return func(res http.ResponseWriter, req *http.Request, params httprouter.Params) {
		LogInfo().Println("Calling jwt middleware..")

		tokenString := req.Header.Get("x-jwt-token")
		token, err := validateJWT(tokenString)
		if err != nil {
			LogError().Println("Permisson denied!") // TODO: Redirect to the error page
			return
		}
		if !token.Valid {
			LogError().Println("Permisson denied!") // TODO: Redirect to the error page
			return
		}
	}
}
