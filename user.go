package main

// TODO: Use datecreated
type User struct {
	Id                ObjectID `bson:"_id"`
	Name              string   `bson:"name"`
	Email             string   `bson:"email"`
	EncryptedPassword string   `bson:"password"`
	Admin             bool     `bson:"admin"`
	DateCreated       string   `bson:"datecreated"`
}

// User authentications
// User permissions
