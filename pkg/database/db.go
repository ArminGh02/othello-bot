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

func (doc *PlayerDoc) String(rank int) string {
	winPercentage := 0
	if matches := doc.Wins + doc.Draws + doc.Losses; matches > 0 {
		winPercentage = int(100 * float64(doc.Wins) / float64(matches))
	}
	return fmt.Sprintf(
		"%s's Profile:\nRank: %d\nWins: %d\nLosses: %d\nDraws: %d\nWin Percentage: %d%%",
		doc.Name,
		rank,
		doc.Wins,
		doc.Losses,
		doc.Draws,
		winPercentage,
	)
}

func (doc *PlayerDoc) Score() int {
	return 3*doc.Wins - doc.Losses
}

type Handler struct {
	client *mongo.Client
	coll   *mongo.Collection
}

func New(uri string) *Handler {
	client, err := mongo.Connect(context.TODO(), options.Client().ApplyURI(uri))
	if err != nil {
		log.Panicln(err)
	}
	coll := client.Database("othello_bot").Collection("players")

	defer log.Println("Connected to MongoDB.")

	return &Handler{
		client: client,
		coll:   coll,
	}
}

func (db *Handler) AddPlayer(userID int64, name string) (added bool) {
	err := db.coll.FindOne(context.TODO(), bson.D{{"user_id", userID}}).Err()
	if err != mongo.ErrNoDocuments {
		return false
	}

	doc := &PlayerDoc{
		UserID:             userID,
		Name:               name,
		Wins:               0,
		Losses:             0,
		Draws:              0,
		LegalMovesAreShown: true,
	}
	_, err = db.coll.InsertOne(context.TODO(), doc)
	if err != nil {
		log.Panicln(err)
	}
	return true
}

func (db *Handler) GetAllPlayers() []PlayerDoc {
	cur, err := db.coll.Find(context.TODO(), bson.D{})
	handleErr(err)
	res := make([]PlayerDoc, 0)
	var doc PlayerDoc
	for cur.Next(context.TODO()) {
		err := cur.Decode(&doc)
		if err != nil {
			log.Panicln(err)
		}
		res = append(res, doc)
	}
	return res
}

func (db *Handler) UsersCount() int64 {
	count, err := db.coll.CountDocuments(context.TODO(), bson.D{})
	if err != nil {
		log.Panicln(err)
	}
	return count
}

func (db *Handler) LegalMovesAreShown(userID int64) bool {
	return db.Find(userID).LegalMovesAreShown
}

func (db *Handler) ToggleLegalMovesAreShown(userID int64) {
	update := bson.D{
		{"$set", bson.D{
			{"legal_moves_are_shown", !db.LegalMovesAreShown(userID)},
		}},
	}
	_, err := db.coll.UpdateOne(context.TODO(), bson.D{{"user_id", userID}}, update)
	handleErr(err)
}

func (db *Handler) IncrementWins(userID int64) {
	db.incrementProperty("wins", userID)
}

func (db *Handler) IncrementLosses(userID int64) {
	db.incrementProperty("losses", userID)
}

func (db *Handler) IncrementDraws(userID int64) {
	db.incrementProperty("draws", userID)
}

func (db *Handler) incrementProperty(propertyName string, userID int64) {
	update := bson.D{
		{"$inc", bson.D{
			{propertyName, 1},
		}},
	}
	_, err := db.coll.UpdateOne(context.TODO(), bson.D{{"user_id", userID}}, update)
	handleErr(err)
}

func (db *Handler) Find(userID int64) *PlayerDoc {
	var doc PlayerDoc
	err := db.coll.FindOne(context.TODO(), bson.D{{"user_id", userID}}).Decode(&doc)
	handleErr(err)
	return &doc
}

func (db *Handler) Disconnect() {
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
