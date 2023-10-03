package main

type User struct {
	Name              string `json:"name"`
	Email             string `json:"email"`
	EncryptedPassword string `json:"password"`
	Admin             bool   `json:"admin"`
	DateOfBirth       string `json:"dateOfBirth"`
	DateCreated       string `json:"datecreated"`
}

type UserLoginRequest struct {
	Email    string
	Password string
}

type UserSignupRequest struct {
	Name        string
	Email       string
	DateOfBirth string
	Password    string
}
