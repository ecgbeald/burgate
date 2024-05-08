package main

import (
	"context"
	"encoding/json"
	"log"
	"net"

	pb "github.com/ecgbeald/burgate/proto"
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

	err = ch.ExchangeDeclare(
		"paid",
		"fanout",
		true,
		false,
		false,
		false,
		nil,
	)
	failOnError(err, "Failed to declare a queue")

	q, err := ch.QueueDeclare(
		"",
		false,
		false,
		true,
		false,
		nil,
	)
	failOnError(err, "Failed to delare a queue")

	log.Printf("Binding queue %s to exchange %s with no route key", q.Name, "paid")
	err = ch.QueueBind(
		q.Name,
		"",
		"paid",
		false,
		nil,
	)
	failOnError(err, "Failed to bind a queue")

	msgs, err := ch.Consume(
		q.Name,
		"",
		false,
		false,
		false,
		false,
		nil,
	)
	failOnError(err, "Failed to register a consumer")

	var forever chan struct{}

	go func() {
		for d := range msgs {
			var dat *pb.Order
			err := json.Unmarshal(d.Body, &dat)
			if err != nil {
				log.Print("Received message cannot be parsed by JSON: ", err)
				continue
			}
			log.Printf("Received a message: %s", dat)
			d.Ack(false)
		}
	}()

	store := NewStore()
	service := NewOrderService(store, ch)

	service.CreateOrder(context.Background())

	NewGRPCHandler(grpcServer, service, ch)

	log.Printf("GRPC server started at 8889")

	if err := grpcServer.Serve(l); err != nil {
		log.Fatal(err.Error())
	}

	log.Printf("[*] Waiting...")
	<-forever
}
