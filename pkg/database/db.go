package database

import (
	"context"
	"log"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type DBHandler struct {
	client *mongo.Client
	coll   *mongo.Collection
}

func New(uri string) *DBHandler {
	client, err := mongo.Connect(context.TODO(), options.Client().ApplyURI(uri))
	if err != nil {
		log.Fatal(err)
	}
	coll := client.Database("othello_bot").Collection("players")
	return &DBHandler{
		client: client,
		coll:   coll,
	}
}
