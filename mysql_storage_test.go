package main

import "testing"

func TestMySqlStorageConnect(t *testing.T) {
	s := NewMySqlStorage()

	err := Connect(s)

	if err != nil {
		t.Errorf("Cant connect to the mysql database, %s", err.Error())
	}

	Disconnect(s)
}

func TestCreateUser(t *testing.T) {
	storage := NewMySqlStorage()

	user := new(User)
	user.Name = "test user"
	user.Email = "test@email.com"
	user.EncryptedPassword = "abcd1234"
	user.Admin = false
	user.DateOfBirth = "2001-06-24"
	user.DateCreated = "17-06-2023"

	err := storage.CheckUserTable()
	if err != nil {
		t.Errorf("Can't create user table, %s", err.Error())
	}

	err = storage.CreateUser(user)
	if err != nil {
		t.Errorf("Can't create user, %s", err.Error())
	}
}
