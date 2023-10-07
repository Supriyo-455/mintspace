package main

import "testing"

func TestMySqlStorageConnect(t *testing.T) {
	s := NewMySqlStorage()

	err := s.Connect()
	defer s.Disconnect()

	if err != nil {
		t.Errorf("Cant connect to the mysql database, %s\n", err.Error())
	}

}

func TestCreateUser(t *testing.T) {
	storage := NewMySqlStorage()

	storage.Connect()
	defer storage.Disconnect()

	user := new(User)
	user.Name = "test user"
	user.Email = "test@email.com"
	user.EncryptedPassword = "abcd1234"
	user.Admin = false
	user.DateOfBirth = "2001-06-24"
	user.DateCreated = "2001-06-24"

	err := storage.CheckUserTable()
	if err != nil {
		t.Errorf("Can't create user table, %s\n", err.Error())
	}

	err = storage.CreateUser(user)
	if err != nil {
		t.Errorf("Can't create user, %s\n", err.Error())
	}
}

func TestDeleteUser(t *testing.T) {
	storage := NewMySqlStorage()

	storage.Connect()
	defer storage.Disconnect()

	err := storage.DeleteUser("test@email.com")
	if err != nil {
		t.Errorf("can't delete user: %s\n", err.Error())
	}
}

func TestCreateAndDeleteBlog(t *testing.T) {
	Storage := NewMySqlStorage()

	Storage.Connect()
	defer Storage.Disconnect()

	blog := Blog{}

	blog.AuthorEmail = "test@author.com"
	blog.CoverImageURL = "sample.com"
	blog.DateCreated = "2020-06-12"
	blog.Premium = false
	blog.Title = "sample blog"
	blog.Content = "Sample content"

	id, err := Storage.CreateBlog(&blog)
	if err != nil {
		t.Errorf("unable to create blog: %s\n", err.Error())
	}

	err = Storage.DeleteBlog(id)
	if err != nil {
		t.Errorf("unable to delete blog: %s\n", err.Error())
	}
}
