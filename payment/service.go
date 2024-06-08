package main

import (
	"context"
	"encoding/json"
	"flag"
	"log"
	"time"

	pb "github.com/ecgbeald/burgate/proto"
	"github.com/ecgbeald/burgate/stripe"
	amqp "github.com/rabbitmq/amqp091-go"
)

func failOnError(err error, msg string) {
	if err != nil {
		log.Panicf("%s: %s", msg, err)
	}
}

func main() {
	status := flag.Bool("local", false, "toggle for local deployment")
	flag.Parse()
	if *status {
		log.Print("Running Locally")
	}
	var conn *amqp.Connection
	var err error
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
		"order",
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

	log.Printf("Binding queue %s to exchange %s with routing key orderch", q.Name, "logs_direct")
	err = ch.QueueBind(
		q.Name,
		"orderch",
		"order",
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
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		for d := range msgs {

			var order *pb.Order
			err := json.Unmarshal(d.Body, &order)
			if err != nil {
				log.Print("Received message cannot be parsed by JSON: ", err)
				continue
			}
			log.Printf("Received a message: %s", order)
			url, err := stripe.CreatePaymentLink(order.Items)
			if err != nil {
				log.Print("Error from Stripe: ", err)
				continue
			}
			log.Println("Payment URL: ", url)
			log.Printf("Waiting for payment...")
			time.Sleep(4 * time.Second)
			order.Status = "Paid"

			d.Ack(false)

			err = ch.ExchangeDeclare("paid", "direct", true, false, false, false, nil)
			failOnError(err, "failed to declare paid exchange")

			body, err := json.Marshal(order)
			if err != nil {
				failOnError(err, "failed to marshal JSON")
			}

			err = ch.PublishWithContext(ctx, "paid", order.GetOrderMachineID(), false, false, amqp.Publishing{
				ContentType: "application/json",
				Body:        body,
			})
			failOnError(err, "failed to publish a message to paid exchange")
			log.Printf(" [x] Sent %s \n", body)
		}
	}()

	log.Printf("[*] Waiting for RPC requests...")
	<-forever
}
