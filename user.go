package main

// TODO: Use datecreated
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
	Email             string `bson:"email"`
	EncryptedPassword string `bson:"password"`
}

type UserSignupRequest struct {
	Name              string `bson:"name"`
	Email             string `bson:"email"`
	DateOfBirth       string `bson:"dateOfBirth"`
	EncryptedPassword string `bson:"password"`
}

// User authentications
// User permissions
