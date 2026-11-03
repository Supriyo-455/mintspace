package main

import (
	"database/sql"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	_ "modernc.org/sqlite"
)

type SqliteStorage struct {
	db     *sql.DB
	dbPath string
}

func NewSqliteStorage(dbPath string) *SqliteStorage {
	return &SqliteStorage{dbPath: dbPath}
}

func (s *SqliteStorage) Connect() error {
	dir := filepath.Dir(s.dbPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create database directory: %w", err)
	}

	// SQLite connection string with foreign keys enabled and WAL mode for fast concurrency
	dsn := fmt.Sprintf("%s?_pragma=busy_timeout(5000)&_pragma=foreign_keys(1)&_pragma=journal_mode(WAL)", s.dbPath)
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return err
	}

	db.SetMaxOpenConns(1) // SQLite works best with 1 writer connection
	db.SetMaxIdleConns(1)
	db.SetConnMaxLifetime(time.Hour)

	if err := db.Ping(); err != nil {
		return err
	}

	s.db = db
	return s.InitSchema()
}

func (s *SqliteStorage) Disconnect() {
	if s.db != nil {
		s.db.Close()
	}
}

func (s *SqliteStorage) InitSchema() error {
	schema := `
	CREATE TABLE IF NOT EXISTS users (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT NOT NULL,
		email TEXT NOT NULL UNIQUE,
		password TEXT NOT NULL,
		admin BOOLEAN NOT NULL DEFAULT 0,
		date_of_birth TEXT NOT NULL DEFAULT '',
		date_created TEXT NOT NULL
	);

	CREATE TABLE IF NOT EXISTS blogs (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		author_email TEXT NOT NULL,
		title TEXT NOT NULL,
		abstract TEXT NOT NULL DEFAULT '',
		cover_image_url TEXT NOT NULL DEFAULT '',
		content_hash TEXT NOT NULL,
		is_private BOOLEAN NOT NULL DEFAULT 0,
		date_created TEXT NOT NULL,
		FOREIGN KEY (author_email) REFERENCES users(email) ON DELETE CASCADE
	);

	CREATE TABLE IF NOT EXISTS likes (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		blog_id INTEGER NOT NULL,
		user_email TEXT NOT NULL,
		created_at TEXT NOT NULL,
		UNIQUE(blog_id, user_email),
		FOREIGN KEY (blog_id) REFERENCES blogs(id) ON DELETE CASCADE,
		FOREIGN KEY (user_email) REFERENCES users(email) ON DELETE CASCADE
	);

	CREATE TABLE IF NOT EXISTS comments (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		blog_id INTEGER NOT NULL,
		parent_id INTEGER NOT NULL DEFAULT 0,
		user_email TEXT NOT NULL,
		author_name TEXT NOT NULL,
		content TEXT NOT NULL,
		created_at TEXT NOT NULL,
		FOREIGN KEY (blog_id) REFERENCES blogs(id) ON DELETE CASCADE,
		FOREIGN KEY (user_email) REFERENCES users(email) ON DELETE CASCADE
	);

	CREATE TABLE IF NOT EXISTS comment_likes (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		comment_id INTEGER NOT NULL,
		user_email TEXT NOT NULL,
		created_at TEXT NOT NULL,
		UNIQUE(comment_id, user_email),
		FOREIGN KEY (comment_id) REFERENCES comments(id) ON DELETE CASCADE,
		FOREIGN KEY (user_email) REFERENCES users(email) ON DELETE CASCADE
	);
	`

	_, err := s.db.Exec(schema)
	if err != nil {
		return fmt.Errorf("failed to init database schema: %w", err)
	}

	// Auto-migrate parent_id if comments table was created prior to threaded discussions
	rows, err := s.db.Query("PRAGMA table_info(comments)")
	if err == nil {
		hasParentId := false
		for rows.Next() {
			var cid, notnull, pk int
			var name, ctype string
			var dfltValue interface{}
			if err := rows.Scan(&cid, &name, &ctype, &notnull, &dfltValue, &pk); err == nil {
				if name == "parent_id" {
					hasParentId = true
					break
				}
			}
		}
		rows.Close()
		if !hasParentId {
			_, _ = s.db.Exec("ALTER TABLE comments ADD COLUMN parent_id INTEGER NOT NULL DEFAULT 0")
		}
	}

	// Now that tables and migrated columns are verified, create indexes
	indexes := `
	CREATE INDEX IF NOT EXISTS idx_blogs_author ON blogs(author_email);
	CREATE INDEX IF NOT EXISTS idx_likes_blog ON likes(blog_id);
	CREATE INDEX IF NOT EXISTS idx_comments_blog ON comments(blog_id);
	CREATE INDEX IF NOT EXISTS idx_comments_parent ON comments(parent_id);
	CREATE INDEX IF NOT EXISTS idx_comment_likes_comment ON comment_likes(comment_id);
	`
	_, _ = s.db.Exec(indexes)

	// Seed default admin if no admin exists
	var count int
	err = s.db.QueryRow("SELECT COUNT(*) FROM users WHERE admin = 1").Scan(&count)
	if err == nil && count == 0 {
		hashedPwd, _ := HashPassword("admin123")
		adminUser := &User{
			Name:              "Editorial Admin",
			Email:             "admin@mintspace.io",
			EncryptedPassword: hashedPwd,
			Admin:             true,
			DateOfBirth:       "1990-01-01",
			DateCreated:       time.Now().Format("2006-01-02"),
		}
		_ = s.CreateUser(adminUser)
		LogInfo().Println("Created default admin: admin@mintspace.io (password: admin123)")
	}

	return nil
}

// ----------------- USER METHODS -----------------

func (s *SqliteStorage) CreateUser(user *User) error {
	query := "INSERT INTO users (name, email, password, admin, date_of_birth, date_created) VALUES (?, ?, ?, ?, ?, ?)"
	_, err := s.db.Exec(query, user.Name, user.Email, user.EncryptedPassword, user.Admin, user.DateOfBirth, user.DateCreated)
	return err
}

func (s *SqliteStorage) DeleteUser(email string) error {
	_, err := s.db.Exec("DELETE FROM users WHERE email = ?", email)
	return err
}

func (s *SqliteStorage) GetUserByEmail(email string) (*User, error) {
	query := "SELECT id, name, email, password, admin, date_of_birth, date_created FROM users WHERE email = ?"
	row := s.db.QueryRow(query, email)

	var user User
	err := row.Scan(&user.Id, &user.Name, &user.Email, &user.EncryptedPassword, &user.Admin, &user.DateOfBirth, &user.DateCreated)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("user not found")
		}
		return nil, err
	}
	return &user, nil
}

func (s *SqliteStorage) GetAllUsers() ([]User, error) {
	rows, err := s.db.Query("SELECT id, name, email, admin, date_of_birth, date_created FROM users ORDER BY id DESC")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var users []User
	for rows.Next() {
		var u User
		if err := rows.Scan(&u.Id, &u.Name, &u.Email, &u.Admin, &u.DateOfBirth, &u.DateCreated); err != nil {
			return nil, err
		}
		users = append(users, u)
	}
	return users, rows.Err()
}

func (s *SqliteStorage) SetUserAdmin(email string, isAdmin bool) error {
	_, err := s.db.Exec("UPDATE users SET admin = ? WHERE email = ?", isAdmin, email)
	return err
}

// ----------------- BLOG METHODS -----------------

func (s *SqliteStorage) CreateBlog(blog *Blog) (int64, error) {
	query := "INSERT INTO blogs (author_email, title, abstract, cover_image_url, content_hash, is_private, date_created) VALUES (?, ?, ?, ?, ?, ?, ?)"
	res, err := s.db.Exec(query, blog.AuthorEmail, blog.Title, blog.Abstract, blog.CoverImageURL, blog.ContentHash, blog.IsPrivate, blog.DateCreated)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

func (s *SqliteStorage) DeleteBlog(id int64) error {
	_, err := s.db.Exec("DELETE FROM blogs WHERE id = ?", id)
	return err
}

func (s *SqliteStorage) GetBlogById(id int64) (*Blog, error) {
	query := `
		SELECT b.id, b.author_email, COALESCE(u.name, b.author_email), b.title, b.abstract, b.cover_image_url, b.content_hash, b.is_private, b.date_created,
		       (SELECT COUNT(*) FROM likes WHERE blog_id = b.id) AS likes_count,
		       (SELECT COUNT(*) FROM comments WHERE blog_id = b.id) AS comments_count
		FROM blogs b
		LEFT JOIN users u ON b.author_email = u.email
		WHERE b.id = ?
	`
	row := s.db.QueryRow(query, id)

	var b Blog
	err := row.Scan(&b.Id, &b.AuthorEmail, &b.AuthorName, &b.Title, &b.Abstract, &b.CoverImageURL, &b.ContentHash, &b.IsPrivate, &b.DateCreated, &b.LikesCount, &b.CommentsCount)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("paper not found")
		}
		return nil, err
	}
	return &b, nil
}

func (s *SqliteStorage) GetAllBlogs(viewerEmail string, isAdmin bool) ([]Blog, error) {
	// If admin, show all. If user, show public + own private. If anonymous, show public only.
	var query string
	var args []interface{}

	if isAdmin {
		query = `
			SELECT b.id, b.author_email, COALESCE(u.name, b.author_email), b.title, b.abstract, b.cover_image_url, b.content_hash, b.is_private, b.date_created,
			       (SELECT COUNT(*) FROM likes WHERE blog_id = b.id) AS likes_count,
			       (SELECT COUNT(*) FROM comments WHERE blog_id = b.id) AS comments_count
			FROM blogs b
			LEFT JOIN users u ON b.author_email = u.email
			ORDER BY b.id DESC
		`
	} else if viewerEmail != "" {
		query = `
			SELECT b.id, b.author_email, COALESCE(u.name, b.author_email), b.title, b.abstract, b.cover_image_url, b.content_hash, b.is_private, b.date_created,
			       (SELECT COUNT(*) FROM likes WHERE blog_id = b.id) AS likes_count,
			       (SELECT COUNT(*) FROM comments WHERE blog_id = b.id) AS comments_count
			FROM blogs b
			LEFT JOIN users u ON b.author_email = u.email
			WHERE b.is_private = 0 OR b.author_email = ?
			ORDER BY b.id DESC
		`
		args = append(args, viewerEmail)
	} else {
		query = `
			SELECT b.id, b.author_email, COALESCE(u.name, b.author_email), b.title, b.abstract, b.cover_image_url, b.content_hash, b.is_private, b.date_created,
			       (SELECT COUNT(*) FROM likes WHERE blog_id = b.id) AS likes_count,
			       (SELECT COUNT(*) FROM comments WHERE blog_id = b.id) AS comments_count
			FROM blogs b
			LEFT JOIN users u ON b.author_email = u.email
			WHERE b.is_private = 0
			ORDER BY b.id DESC
		`
	}

	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var blogs []Blog
	for rows.Next() {
		var b Blog
		if err := rows.Scan(&b.Id, &b.AuthorEmail, &b.AuthorName, &b.Title, &b.Abstract, &b.CoverImageURL, &b.ContentHash, &b.IsPrivate, &b.DateCreated, &b.LikesCount, &b.CommentsCount); err != nil {
			return nil, err
		}
		blogs = append(blogs, b)
	}
	return blogs, rows.Err()
}

func (s *SqliteStorage) GetBlogsByAuthor(authorEmail string) ([]Blog, error) {
	query := `
		SELECT b.id, b.author_email, COALESCE(u.name, b.author_email), b.title, b.abstract, b.cover_image_url, b.content_hash, b.is_private, b.date_created,
		       (SELECT COUNT(*) FROM likes WHERE blog_id = b.id) AS likes_count,
		       (SELECT COUNT(*) FROM comments WHERE blog_id = b.id) AS comments_count
		FROM blogs b
		LEFT JOIN users u ON b.author_email = u.email
		WHERE b.author_email = ?
		ORDER BY b.id DESC
	`
	rows, err := s.db.Query(query, authorEmail)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var blogs []Blog
	for rows.Next() {
		var b Blog
		if err := rows.Scan(&b.Id, &b.AuthorEmail, &b.AuthorName, &b.Title, &b.Abstract, &b.CoverImageURL, &b.ContentHash, &b.IsPrivate, &b.DateCreated, &b.LikesCount, &b.CommentsCount); err != nil {
			return nil, err
		}
		blogs = append(blogs, b)
	}
	return blogs, rows.Err()
}

// ----------------- LIKE METHODS -----------------

func (s *SqliteStorage) ToggleLike(blogId int64, userEmail string) (bool, int, error) {
	hasLiked, err := s.HasUserLiked(blogId, userEmail)
	if err != nil {
		return false, 0, err
	}

	if hasLiked {
		_, err = s.db.Exec("DELETE FROM likes WHERE blog_id = ? AND user_email = ?", blogId, userEmail)
		if err != nil {
			return false, 0, err
		}
		hasLiked = false
	} else {
		_, err = s.db.Exec("INSERT INTO likes (blog_id, user_email, created_at) VALUES (?, ?, ?)", blogId, userEmail, time.Now().Format("2006-01-02 15:04:05"))
		if err != nil {
			return false, 0, err
		}
		hasLiked = true
	}

	var count int
	_ = s.db.QueryRow("SELECT COUNT(*) FROM likes WHERE blog_id = ?", blogId).Scan(&count)
	return hasLiked, count, nil
}

func (s *SqliteStorage) HasUserLiked(blogId int64, userEmail string) (bool, error) {
	var count int
	err := s.db.QueryRow("SELECT COUNT(*) FROM likes WHERE blog_id = ? AND user_email = ?", blogId, userEmail).Scan(&count)
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

// ----------------- COMMENT METHODS -----------------

func (s *SqliteStorage) CreateComment(comment *Comment) (int64, error) {
	query := "INSERT INTO comments (blog_id, parent_id, user_email, author_name, content, created_at) VALUES (?, ?, ?, ?, ?, ?)"
	res, err := s.db.Exec(query, comment.BlogId, comment.ParentId, comment.UserEmail, comment.AuthorName, comment.Content, comment.DateCreated)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

func (s *SqliteStorage) ToggleCommentLike(commentId int64, userEmail string) (bool, int, error) {
	var count int
	err := s.db.QueryRow("SELECT COUNT(*) FROM comment_likes WHERE comment_id = ? AND user_email = ?", commentId, userEmail).Scan(&count)
	if err != nil {
		return false, 0, err
	}

	hasLiked := false
	if count > 0 {
		_, err = s.db.Exec("DELETE FROM comment_likes WHERE comment_id = ? AND user_email = ?", commentId, userEmail)
		if err != nil {
			return false, 0, err
		}
	} else {
		_, err = s.db.Exec("INSERT INTO comment_likes (comment_id, user_email, created_at) VALUES (?, ?, ?)", commentId, userEmail, time.Now().Format("2006-01-02 15:04:05"))
		if err != nil {
			return false, 0, err
		}
		hasLiked = true
	}

	var newCount int
	_ = s.db.QueryRow("SELECT COUNT(*) FROM comment_likes WHERE comment_id = ?", commentId).Scan(&newCount)
	return hasLiked, newCount, nil
}

func (s *SqliteStorage) GetCommentsByBlogId(blogId int64, viewerEmail string) ([]Comment, error) {
	query := `
		SELECT c.id, c.blog_id, c.parent_id, c.user_email, c.author_name, c.content, c.created_at,
		       (SELECT COUNT(*) FROM comment_likes WHERE comment_id = c.id) AS likes_count,
		       (SELECT COUNT(*) FROM comment_likes WHERE comment_id = c.id AND user_email = ?) AS user_has_liked
		FROM comments c
		WHERE c.blog_id = ?
		ORDER BY c.id ASC
	`
	rows, err := s.db.Query(query, viewerEmail, blogId)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var allComments []Comment
	commentMap := make(map[int64]*Comment)

	for rows.Next() {
		var c Comment
		var userHasLikedInt int
		if err := rows.Scan(&c.Id, &c.BlogId, &c.ParentId, &c.UserEmail, &c.AuthorName, &c.Content, &c.DateCreated, &c.LikesCount, &userHasLikedInt); err != nil {
			return nil, err
		}
		c.UserHasLiked = userHasLikedInt > 0
		c.Replies = []Comment{}
		allComments = append(allComments, c)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	for i := range allComments {
		commentMap[allComments[i].Id] = &allComments[i]
	}

	var rootComments []Comment
	var assemble func(id int64) Comment
	assemble = func(id int64) Comment {
		node := *commentMap[id]
		node.Replies = nil
		for _, item := range allComments {
			if item.ParentId == id {
				node.Replies = append(node.Replies, assemble(item.Id))
			}
		}
		return node
	}

	for _, c := range allComments {
		if c.ParentId == 0 {
			rootComments = append(rootComments, assemble(c.Id))
		}
	}

	return rootComments, nil
}

func (s *SqliteStorage) GetCommentById(id int64) (*Comment, error) {
	var c Comment
	err := s.db.QueryRow("SELECT id, blog_id, parent_id, user_email, author_name, content, created_at FROM comments WHERE id = ?", id).
		Scan(&c.Id, &c.BlogId, &c.ParentId, &c.UserEmail, &c.AuthorName, &c.Content, &c.DateCreated)
	if err != nil {
		return nil, err
	}
	return &c, nil
}

func (s *SqliteStorage) DeleteComment(id int64) error {
	_, _ = s.db.Exec("DELETE FROM comments WHERE parent_id = ?", id)
	_, _ = s.db.Exec("DELETE FROM comment_likes WHERE comment_id = ? OR comment_id IN (SELECT id FROM comments WHERE parent_id = ?)", id, id)
	_, err := s.db.Exec("DELETE FROM comments WHERE id = ?", id)
	return err
}

// ----------------- STATS -----------------

func (s *SqliteStorage) GetStats() (*Stats, error) {
	stats := &Stats{}
	_ = s.db.QueryRow("SELECT COUNT(*) FROM users").Scan(&stats.TotalUsers)
	_ = s.db.QueryRow("SELECT COUNT(*) FROM blogs").Scan(&stats.TotalBlogs)
	_ = s.db.QueryRow("SELECT COUNT(*) FROM blogs WHERE is_private = 0").Scan(&stats.TotalPublic)
	_ = s.db.QueryRow("SELECT COUNT(*) FROM blogs WHERE is_private = 1").Scan(&stats.TotalPrivate)
	_ = s.db.QueryRow("SELECT COUNT(*) FROM comments").Scan(&stats.TotalComments)
	_ = s.db.QueryRow("SELECT COUNT(*) FROM likes").Scan(&stats.TotalLikes)
	return stats, nil
}
