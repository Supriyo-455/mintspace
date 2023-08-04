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

	if id != user.Id {
		t.Errorf("want id=%s but got id=%s\n", id, user.Id)
	}

	path := fmt.Sprintln("blogs/", id)
	_, err = os.Stat(path)
	if os.IsNotExist(err) {
		t.Errorf("error occured: %s", err.Error())
	}
}
