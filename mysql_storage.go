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

func (s *MySqlStorage) Connect() error {
	db, err := sql.Open(s.config.Driver, s.config.User+":"+s.config.Password+"@"+s.config.Database)
	if err != nil {
		return err
	}

	s.db = db
	return nil
}

func (s *MySqlStorage) Disconnect() {
	s.db.Close()
}

func (s *MySqlStorage) CheckUserTable() error {
	query, err := s.db.Prepare("create table if not exists user (name varchar(50) not null, email varchar(50) not null unique, password varchar(256) not null, admin bool not null, dateOfBirth date not null, datecreated date not null, primary key(email))")
	if err != nil {
		return err
	}

	query.Exec()
	query.Close()
	return nil
}

func (s *MySqlStorage) CreateUser(user *User) error {
	err := s.CheckUserTable()
	if err != nil {
		return err
	}

	queryString := "insert into user (name, email, password, admin, dateOfBirth, dateCreated) values(?, ?, ?, ?, ?, ?)"
	res, err := s.db.Exec(queryString, user.Name, user.Email, user.EncryptedPassword, user.Admin, user.DateOfBirth, user.DateCreated)
	if err != nil {
		return err
	}

	rowCnt, err := res.RowsAffected()
	if err != nil {
		return err
	}
	LogInfo().Println("Rows affected: ", rowCnt)

	return nil
}

func (s *MySqlStorage) GetUserByEmail(email string) (*User, error) {
	err := s.CheckUserTable()
	if err != nil {
		return nil, err
	}

	rows, err := s.db.Query("select * from user where email=?", email)
	if err != nil {
		return nil, err
	}

	user := new(User)
	// Improve this variable scanning?
	if rows.Next() {
		var name string
		var email string
		var passwordHash string
		var admin bool
		var dateOfBirth string
		var dateCreated string
		err = rows.Scan(&name, &email, &passwordHash, &admin, &dateOfBirth, &dateCreated)
		if err != nil {
			return nil, err
		}

		user.Name = name
		user.Email = email
		user.EncryptedPassword = passwordHash
		user.Admin = admin
		user.DateOfBirth = dateOfBirth
		user.DateCreated = dateCreated
	}

	return user, nil
}

func (s *MySqlStorage) CreateBlog(blog *Blog) error {
	queryString := "insert into blog (id, author, title, imageurl, premium, dateCreated) values(?, ?, ?, ?, ?, ?)"
	res, err := s.db.Exec(queryString, blog.Id, blog.Author, blog.Title, blog.CoverImageURL, blog.Premium, blog.DateCreated)
	if err != nil {
		return err
	}

	rowCnt, err := res.RowsAffected()
	if err != nil {
		return err
	}
	LogInfo().Println("Rows affected: ", rowCnt)

	return nil
}
