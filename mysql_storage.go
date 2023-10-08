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

func (s *MySqlStorage) DeleteUser(email string) error {
	err := s.CheckUserTable()
	if err != nil {
		return err
	}

	queryString := "delete from user where email=?"
	_, err = s.db.Exec(queryString, email)
	if err != nil {
		return err
	}

	LogInfo().Println("Deletion sucessful for email id: ", email)
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

func (s *MySqlStorage) CheckBlogTable() error {
	queryString := "create table if not exists blog (id integer not null auto_increment, authorEmail varchar(156) not null, title varchar(156) not null, coverImageURL text, content text, premium bool not null, dateCreated date not null, primary key(id))"
	_, err := s.db.Exec(queryString)
	if err != nil {
		return err
	}

	LogInfo().Println("Blog table created!")
	return nil
}

func (s *MySqlStorage) CreateBlog(blog *Blog) (int64, error) {
	err := s.CheckBlogTable()
	if err != nil {
		return 0, err
	}

	queryString := "insert into blog (authorEmail, title, coverImageURL, content, premium, dateCreated) values(?, ?, ?, ?, ?, ?)"
	res, err := s.db.Exec(queryString, blog.AuthorEmail, blog.Title, blog.CoverImageURL, blog.Content, blog.Premium, blog.DateCreated)
	if err != nil {
		return 0, err
	}

	blogId, err := res.LastInsertId()
	if err != nil {
		return 0, err
	}

	rowCnt, err := res.RowsAffected()
	if err != nil {
		return 0, err
	}
	LogInfo().Println("Rows affected: ", rowCnt)

	return blogId, nil
}

func (s *MySqlStorage) DeleteBlog(id int64) error {
	err := s.CheckBlogTable()
	if err != nil {
		return err
	}

	queryString := "delete from blog where id=?"
	_, err = s.db.Exec(queryString, id)
	if err != nil {
		return err
	}

	LogInfo().Println("Deletion successful of blog with id: ", id)
	return nil
}

func (s *MySqlStorage) GetAllBlogs() (*Blogs, error) {
	err := s.CheckBlogTable()
	if err != nil {
		return nil, err
	}

	rows, err := s.db.Query("select * from blog")
	if err != nil {
		return nil, err
	}

	defer rows.Close()

	blogs := new(Blogs)

	for rows.Next() {
		var blog Blog
		if err := rows.Scan(&blog.Id, &blog.AuthorEmail, &blog.Title, &blog.CoverImageURL, &blog.Content, &blog.Premium, &blog.DateCreated); err != nil {
			return blogs, err
		}
		blogs.Array = append(blogs.Array, blog)
	}

	if err = rows.Err(); err != nil {
		return blogs, err
	}
	return blogs, err
}
