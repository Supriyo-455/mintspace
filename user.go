package main

type User struct {
	Id                int64  `json:"id"`
	Name              string `json:"name"`
	Email             string `json:"email"`
	EncryptedPassword string `json:"password"`
	Admin             bool   `json:"admin"`
}
