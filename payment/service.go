package main

import (
	"context"
	"encoding/json"
	"log"
	"time"

	pb "github.com/ecgbeald/burgate/proto"
	amqp "github.com/rabbitmq/amqp091-go"
)

func failOnError(err error, msg string) {
	if err != nil {
		log.Panicf("%s: %s", msg, err)
	}
}

func main() {
	conn, err := amqp.Dial("amqp://guest:guest@localhost:5672/")
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

			var dat *pb.Order
			err := json.Unmarshal(d.Body, &dat)
			if err != nil {
				log.Print("Received message cannot be parsed by JSON: ", err)
				continue
			}
			log.Printf("Received a message: %s", dat)
			log.Printf("Waiting for payment...")
			time.Sleep(4 * time.Second)
			dat.Status = "Paid"

			d.Ack(false)

			err = ch.ExchangeDeclare("paid", "fanout", true, false, false, false, nil)
			failOnError(err, "failed to declare paid exchange")

			body, err := json.Marshal(dat)
			if err != nil {
				failOnError(err, "failed to marshal JSON")
			}

			err = ch.PublishWithContext(ctx, "paid", "", false, false, amqp.Publishing{
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
