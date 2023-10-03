package main

import (
	"database/sql"

	_ "github.com/go-sql-driver/mysql"
)

type MySqlConfig struct {
	Driver   string `json:"driver"`
	User     string `json:"user"`
	Database string `json:"database"`
	Password string `json:"password"`
}

type MySqlStorage struct {
	db     *sql.DB
	config MySqlConfig
}

func NewMySqlStorage() *MySqlStorage {
	config := new(MySqlConfig)
	LoadJson("mysqlConfig.json", config)

	storage := new(MySqlStorage)
	storage.config = *config

	return storage
}

func Connect(s *MySqlStorage) error {
	db, err := sql.Open(s.config.Driver, s.config.User+":"+s.config.Password+"@"+s.config.Database)
	if err != nil {
		return err
	}

	s.db = db
	return nil
}

func Disconnect(s *MySqlStorage) {
	s.db.Close()
}

func (s *MySqlStorage) CheckUserTable() error {
	err := Connect(s)
	if err != nil {
		return err
	}

	defer Disconnect(s)

	query, err := s.db.Prepare("create table if not exists user (name varchar(50) not null, email varchar(50) not null unique, password varchar(256) not null, admin bool not null, dateOfBirth date not null, datecreated date not null, primary key(email))")
	if err != nil {
		return err
	}

	query.Exec()
	return nil
}

func (s *MySqlStorage) CreateUser(user *User) error {
	err := Connect(s)
	if err != nil {
		return err
	}

	defer Disconnect(s)

	query, err := s.db.Prepare("insert into user (name, email, password, admin, dateOfBirth, dateCreated) values(?, ?, ?, ?, ?, ?)")
	if err != nil {
		return err
	}
	query.Exec(user.Name, user.Email, user.EncryptedPassword, user.Admin, user.DateOfBirth, user.DateCreated)

	return nil
}

func (s *MySqlStorage) GetUserByEmail(email string) (*User, error) {
	err := Connect(s)
	if err != nil {
		return nil, err
	}

	defer Disconnect(s)

	rows, err := s.db.Query("select * from user where email=?", email)
	if err != nil {
		return nil, err
	}

	user := new(User)
	if rows.Next() {
		err = rows.Scan(user.Name, user.Email, user.EncryptedPassword, user.Admin, user.DateOfBirth, user.DateCreated)
		if err != nil {
			return nil, err
		}
	}

	return user, nil
}

func (s *MySqlStorage) CreateBlog(blog *Blog) error {
	err := Connect(s)
	if err != nil {
		return err
	}

	defer Disconnect(s)

	query, err := s.db.Prepare("insert into blog (id, author, title, imageurl, premium, dateCreated) values(?, ?, ?, ?, ?, ?)")
	if err != nil {
		return err
	}
	query.Exec(blog.Id, blog.Author, blog.Title, blog.CoverImageURL, blog.Premium, blog.DateCreated)

	return nil
}
