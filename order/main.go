package main

import (
	"context"
	"encoding/json"
	"flag"
	"log"
	"net"
	"os"
	"time"

	pb "github.com/ecgbeald/burgate/proto"
	amqp "github.com/rabbitmq/amqp091-go"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"google.golang.org/grpc"
)

func main() {
	status := flag.Bool("local", false, "toggle for local deployment")
	flag.Parse()
	if *status {
		log.Print("Running Locally")
	}
	grpcServer := grpc.NewServer()

	port := os.Getenv("PORT")

	l, err := net.Listen("tcp", ":"+port)
	if err != nil {
		log.Fatal("failed to listen ", err)
	}

	defer l.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var mongoClientOptions *options.ClientOptions
	if *status {
		mongoClientOptions = options.Client().ApplyURI("mongodb://admin:admin@localhost:27017")
	} else {
		mongoClientOptions = options.Client().ApplyURI("mongodb://admin:admin@mongo:27017")
	}

	mongoCli, err := mongo.Connect(ctx, mongoClientOptions)
	defer mongoCli.Disconnect(ctx)
	failOnError(err, "Failed to connect to MongoDB...")

	err = mongoCli.Ping(ctx, nil)
	failOnError(err, "Failed to ping MongoDB")

	log.Println("Connected to MongoDB")

	var conn *amqp.Connection
	if *status {
		conn, err = amqp.Dial("amqp://guest:guest@localhost:5672/")
	} else {
		conn, err = amqp.Dial("amqp://guest:guest@rabbitmq:5672/")
	}
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
		"direct",
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
		os.Getenv("ID"),
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

	store := NewStore(mongoCli)
	service := NewOrderService(store, ch)

	NewGRPCHandler(grpcServer, store, service, ch)

	log.Printf("GRPC server started at %s", port)

	if err := grpcServer.Serve(l); err != nil {
		log.Fatal(err.Error())
	}

	log.Printf("[*] Waiting...")
	<-forever
}
