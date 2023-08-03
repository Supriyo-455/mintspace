package main

import (
	"context"
	"fmt"
	"os"
	"testing"
)

// TODO: seperate mongo test database and production database
func TestInsertUser(t *testing.T) {
	user := User{
		Name:              "test testing",
		Email:             "test@email.com",
		EncryptedPassword: "abcdef",
		Admin:             true,
		DateCreated:       "1000-00-00",
	}

	mongoStorage := createMongoStorage()
	err := mongoStorage.Connect(context.TODO())
	if err != nil {
		t.Errorf("error occured: %s", err.Error())
	}

	id, err := mongoStorage.InsertUser(context.TODO(), &user)
	if err != nil {
		t.Errorf("error occured: %s", err.Error())
	}

	path := fmt.Sprintln("blogs/", id.Hex())
	_, err = os.Stat(path)
	if os.IsNotExist(err) {
		t.Errorf("error occured: %s", err.Error())
	}
}
