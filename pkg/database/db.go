package database

import (
	"context"
	"log"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type PlayerDoc struct {
	UserID             int64
	Name               string
	Wins               int
	Losses             int
	Draws              int
	LegalMovesAreShown bool
}

func newPlayerDoc(userID int64, name string, wins, losses, draws int, legalMovesAreShown bool) *PlayerDoc {
	return &PlayerDoc{
		UserID:             userID,
		Name:               name,
		Wins:               wins,
		Losses:             losses,
		Draws:              draws,
		LegalMovesAreShown: legalMovesAreShown,
	}
}

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

func (db *DBHandler) AddPlayer(userID int64, name string) {
	doc := newPlayerDoc(userID, name, 0, 0, 0, true)
	_, err := db.coll.InsertOne(context.TODO(), doc)
	if err != nil {
		log.Panicln(err)
	}
}
