package main

// TODO: Use datecreated
type User struct {
	Id                interface{} `bson:"_id,omitempty"` // For mapping mongodbs id to golang struct
	Name              string      `bson:"name"`
	Email             string      `bson:"email"`
	EncryptedPassword string      `bson:"password"`
	Admin             bool        `bson:"admin"`
	DateCreated       string      `bson:"datecreated"`
}

// User authentications
// User permissions
