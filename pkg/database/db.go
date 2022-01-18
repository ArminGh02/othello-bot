package database

import (
	"context"
	"fmt"
	"log"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type PlayerDoc struct {
	UserID             int64  `bson:"user_id"`
	Name               string `bson:"name"`
	Wins               int    `bson:"wins"`
	Losses             int    `bson:"losses"`
	Draws              int    `bson:"draws"`
	LegalMovesAreShown bool   `bson:"legal_moves_are_shown"`
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

func (doc *PlayerDoc) String() string {
	return fmt.Sprintf("%s's Profile:\nWins: %d\nLosses: %d\nDraws: %d",
		doc.Name,
		doc.Wins,
		doc.Losses,
		doc.Draws,
	)
}

type DBHandler struct {
	client *mongo.Client
	coll   *mongo.Collection
}

func New(uri string) *DBHandler {
	client, err := mongo.Connect(context.TODO(), options.Client().ApplyURI(uri))
	if err != nil {
		log.Fatalln(err)
	}
	coll := client.Database("othello_bot").Collection("players")

	defer log.Println("Connected to MongoDB")

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

func (db *DBHandler) LegalMovesAreShown(userID int64) bool {
	return db.ProfileOf(userID).LegalMovesAreShown
}

func (db *DBHandler) ToggleLegalMovesAreShown(userID int64) {
	update := bson.D{
		{"$set", bson.D{
			{"legal_moves_are_shown", !db.LegalMovesAreShown(userID)},
		}},
	}
	_, err := db.coll.UpdateOne(context.TODO(), bson.D{{"user_id", userID}}, update)
	handleErr(err)
}

func (db *DBHandler) IncrementWins(userID int64) {
	db.incrementProperty("wins", userID)
}

func (db *DBHandler) IncrementLosses(userID int64) {
	db.incrementProperty("losses", userID)
}

func (db *DBHandler) IncrementDraws(userID int64) {
	db.incrementProperty("draws", userID)
}

func (db *DBHandler) incrementProperty(propertyName string, userID int64) {
	update := bson.D{
		{"$inc", bson.D{
			{propertyName, 1},
		}},
	}
	_, err := db.coll.UpdateOne(context.TODO(), bson.D{{"user_id", userID}}, update)
	handleErr(err)
}

func (db *DBHandler) ProfileOf(userID int64) *PlayerDoc {
	var doc PlayerDoc
	err := db.coll.FindOne(context.TODO(), bson.D{{"user_id", userID}}).Decode(&doc)
	handleErr(err)
	return &doc
}

func (db *DBHandler) Disconnect() {
	if err := db.client.Disconnect(context.TODO()); err != nil {
		log.Panicln(err)
	}
}

func handleErr(err error) {
	if err == mongo.ErrNoDocuments {
		log.Panicln("An attempt was made to retrieve the user that was not inserted.", err)
	}
	if err != nil {
		log.Panicln(err)
	}
}
