package main

import (
	"context"
	"encoding/json"
	"flag"
	"log"
	"os"
	"strconv"
	"time"

	pb "github.com/ecgbeald/burgate/proto"
	amqp "github.com/rabbitmq/amqp091-go"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type handler struct {
	client pb.OrderServiceClient
}

func NewHandler(client pb.OrderServiceClient) *handler {
	return &handler{client: client}
}

func failOnError(err error, msg string) {
	if err != nil {
		log.Panicf("%s: %s", msg, err)
	}
}

func main() {
	var conn *amqp.Connection
	var err error
	status := flag.Bool("local", false, "toggle for local deployment")
	flag.Parse()
	if *status {
		log.Print("Running Locally")
	}

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

	// consumer might exist before the producer, make sure queue exists before consuming msg
	err = ch.ExchangeDeclare(
		"paid",
		"direct",
		true,
		false,
		false,
		false,
		nil,
	)
	failOnError(err, "Failed to declare an exchange")

	q, err := ch.QueueDeclare(
		"",
		false,
		false,
		true,
		false,
		nil,
	)
	failOnError(err, "Failed to delare a queue")

	orderNodeCnt, err := strconv.Atoi(os.Getenv("ORDERNODECNT"))
	if err != nil {
		log.Panicf("Environment variable not parsed correctly: %s", err)
	}
	log.Printf("Binding queue %s to exchange %s with %d route key", q.Name, "paid", orderNodeCnt)

	for i := 0; i < orderNodeCnt; i++ {
		err = ch.QueueBind(
			q.Name,
			strconv.Itoa(i),
			"paid",
			false,
			nil,
		)
		failOnError(err, "Failed to bind a queue")
	}

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
			log.Printf("Cooking...")
			time.Sleep(5 * time.Second)
			d.Ack(false)

			dat.Status = "Finished"

			addr := "order-" + dat.OrderMachineID + ":8889"
			log.Println("Connecting to ", addr)
			conn, err := grpc.Dial(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
			if err != nil {
				log.Fatalln("Failed to connect to", addr, ", error: ", err)
			}
			cli := pb.NewOrderServiceClient(conn)
			handler := NewHandler(cli)
			_, err = handler.client.ReceiveCookedOrder(context.Background(), dat)
			if err != nil {
				log.Panic("Error sending cooked order: ", err)
			}
			conn.Close()
		}
	}()

	log.Printf("[*] Waiting...")
	<-forever
}
