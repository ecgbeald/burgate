package main

import (
	"context"
	"fmt"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

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

func (s *store) Create(context.Context) error {
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
