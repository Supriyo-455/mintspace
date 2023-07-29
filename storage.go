package main

import (
	"context"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"
)

func ConnectToMongo() *mongo.Client {
	client, err := mongo.Connect(context.TODO(), options.Client().ApplyURI("mongodb://localhost:27017"))
	if err != nil {
		Error().Fatalln(err)
	}

	if err := client.Ping(context.TODO(), readpref.Primary()); err != nil {
		Error().Fatalln(err)
	}

	return client
}

func TestMongo(client *mongo.Client) {
	usersCollection := client.Database("testing").Collection("users")

	user := bson.D{{Key: "fullName", Value: "User 1"}, {Key: "age", Value: 30}}

	result, err := usersCollection.InsertOne(context.TODO(), user)
	if err != nil {
		Error().Fatalln(err)
	}
	Info().Println(result.InsertedID)

	users := []interface{}{
		bson.D{{Key: "fullName", Value: "User 2"}, {Key: "age", Value: 25}},
		bson.D{{Key: "fullName", Value: "User 3"}, {Key: "age", Value: 20}},
		bson.D{{Key: "fullName", Value: "User 4"}, {Key: "age", Value: 28}},
	}
	results, err := usersCollection.InsertMany(context.TODO(), users)
	if err != nil {
		Error().Fatalln(err)
	}
	Info().Println(results.InsertedIDs)
}
