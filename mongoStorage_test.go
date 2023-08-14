package main

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

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

	path := filepath.Join(".", "blogs", string(id))
	_, err = os.Stat(path)
	if os.IsNotExist(err) {
		t.Errorf("error occured: %s", err.Error())
	}
}
