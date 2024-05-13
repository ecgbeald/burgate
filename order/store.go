package main

import (
	"context"
	"fmt"
	"log"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

// MENU

type menu_entry_db struct {
	Id       string
	Name     string
	Price_id string
}

type store struct {
	mongoCli *mongo.Client
}

func NewStore(mongoCli *mongo.Client) *store {
	return &store{mongoCli: mongoCli}
}

func (s *store) GetMongoClient() *mongo.Client {
	return s.mongoCli
}

func (s *store) Create(ctx context.Context, menu *[]menu_entry_db) error {
	collection := s.mongoCli.Database("menu").Collection("menu")
	for _, entry := range *menu {
		filter := bson.D{{Key: "id", Value: entry.Id}}
		update := bson.D{{Key: "$set", Value: entry}}
		result, err := collection.UpdateOne(ctx, filter, update)
		if err != nil {
			log.Print("err when updating db: ", err)
		}
		if result.MatchedCount != 0 {
			continue
		}
		_, err = collection.InsertOne(ctx, entry)
		if err != nil {
			log.Print("err when inserting into db: ", err)
		}
		log.Print("result:", result)

	}
	return nil
}

func (s *store) Query(ctx context.Context, id string) (*menu_entry_db, error) {
	collection := s.mongoCli.Database("menu").Collection("menu")
	filter := bson.D{{Key: "id", Value: id}}
	cursor := collection.FindOne(ctx, filter)
	if cursor == nil {
		return nil, fmt.Errorf("not found")
	}
	var result menu_entry_db

	if err := cursor.Decode(&result); err != nil {
		return nil, err
	}

	res, _ := bson.MarshalExtJSON(result, false, false)
	fmt.Println(string(res))
	return &result, nil
}
