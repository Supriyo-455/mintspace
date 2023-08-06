package main

import (
	"context"
	"path/filepath"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type MongoConfig struct {
	URI      string
	Database string
}

type MongoStorage struct {
	ClientOptions *options.ClientOptions
	Client        *mongo.Client
}

const (
	mongoConfigFile = "mongoConfig.json"
	usersCollection = "users"
	blogsCollection = "blogs"
)

var mongoConfiguration MongoConfig

func init() {
	mongoConfiguration = MongoConfig{}
	LoadJson(mongoConfigFile, &mongoConfiguration)
}

/*
	 TODO:
		1) Singleton pattern
		2) error handling?
*/
func createMongoStorage() *MongoStorage {
	storage := new(MongoStorage)
	storage.ClientOptions = options.Client().ApplyURI(mongoConfiguration.URI)

	return storage
}

func (ms *MongoStorage) Connect(ctx context.Context) error {
	var err error
	ms.Client, err = mongo.Connect(ctx, ms.ClientOptions)
	if err != nil {
		return err
	}
	return nil
}

// TODO: check
func (ms *MongoStorage) InsertUser(ctx context.Context, user *User) (ObjectID, error) {
	collection := ms.Client.Database(mongoConfiguration.Database).Collection(usersCollection)
	indexs, err := collection.Indexes().ListSpecifications(ctx)
	if err != nil {
		return "", err
	}

	// Searching for index
	found := false
	for _, index := range indexs {
		if index.Name == "email_1" {
			found = true
		}
	}

	// creating index if not found
	if !found {
		_, err := collection.Indexes().CreateOne(ctx, mongo.IndexModel{
			Keys:    bson.M{"email": 1},
			Options: options.Index().SetUnique(true),
		})
		if err != nil {
			return "", err
		}
	}

	mongoObjectID := primitive.NewObjectID()
	user.Id = ObjectID(mongoObjectID.Hex())

	user.DateCreated = time.Now().Format(time.DateOnly)

	_, err = collection.InsertOne(ctx, user)
	if err != nil {
		return "", err
	}

	path := filepath.Join(".", "blogs", string(user.Id))
	err = CreateFolder(path)
	if err != nil {
		return "", err
	}

	return user.Id, err
}

func (ms *MongoStorage) InsertBlog(ctx context.Context, blog *Blog) (string, error) {
	collection := ms.Client.Database(mongoConfiguration.Database).Collection(blogsCollection)

	mongoObjectID := primitive.NewObjectID()
	blog.Id = ObjectID(mongoObjectID.Hex())
	blog.DateCreated = time.Now().Format(time.DateOnly)

	_, err := collection.InsertOne(ctx, blog)
	if err != nil {
		return "", err
	}

	path := filepath.Join(".", "blogs", string(blog.Author.Id), string(blog.Id)+".md")
	return path, nil
}

// func (ms *MongoStorage) GetAllBlogs(ctx context.Context) (Blogs, error) {
// 	collection := ms.Client.Database(mongoConfiguration.Database).Collection(blogsCollection)
// }
