package main

import (
	"context"
	"log"
	"net"

	amqp "github.com/rabbitmq/amqp091-go"
	"google.golang.org/grpc"
)

func main() {
	grpcServer := grpc.NewServer()

	l, err := net.Listen("tcp", ":8889")
	if err != nil {
		log.Fatal("failed to listen ", err)
	}

	defer l.Close()

	conn, err := amqp.Dial("amqp://guest:guest@localhost:5672/")
	failOnError(err, "Failed to connect to RabbitMQ")
	defer conn.Close()

	ch, err := conn.Channel()
	failOnError(err, "Failed to open a channel")
	defer ch.Close()

	err = ch.ExchangeDeclare(
		"order",
		"direct",
		true,
		false,
		false,
		false,
		nil,
	)
	failOnError(err, "Failed to declare a queue")

	store := NewStore()
	service := NewOrderService(store, ch)

	service.CreateOrder(context.Background())

	NewGRPCHandler(grpcServer, service, ch)

	log.Printf("GRPC server started at 8889")

	if err := grpcServer.Serve(l); err != nil {
		log.Fatal(err.Error())
	}
}
