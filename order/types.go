package main

import (
	"context"

	"go.mongodb.org/mongo-driver/mongo"
)

type OrderService interface {
}

type OrderStore interface {
	Create(context.Context, *[]menu_entry_db) error
	Query(context.Context, string) (*menu_entry_db, error)
	GetMongoClient() *mongo.Client
}
