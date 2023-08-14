package main

type User struct {
	Id                ObjectID `bson:"_id"`
	Name              string   `bson:"name"`
	Email             string   `bson:"email"`
	EncryptedPassword string   `bson:"password"`
	Admin             bool     `bson:"admin"`
	DateOfBirth       string   `bson:"dateOfBirth"`
	DateCreated       string   `bson:"datecreated"`
}

type UserLoginRequest struct {
	Email             string
	EncryptedPassword string
}

type UserSignupRequest struct {
	Name              string
	Email             string
	DateOfBirth       string
	EncryptedPassword string
}
