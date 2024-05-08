package main

import (
	"context"

	amqp "github.com/rabbitmq/amqp091-go"
)

type service struct {
	store    OrderStore
	rabbitch *amqp.Channel
}

func NewOrderService(store OrderStore, ch *amqp.Channel) *service {
	return &service{store: store, rabbitch: ch}
}

func (s *service) CreateOrder(context.Context) error {
	return nil
}
