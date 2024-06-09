package main

import (
	"context"
	"encoding/json"
	"log"
	"os"

	pb "github.com/ecgbeald/burgate/proto"
	stripe "github.com/ecgbeald/burgate/stripe_service/utils"
	"github.com/joho/godotenv"
	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/redis/go-redis/v9"
)

func failOnError(err error, msg string) {
	if err != nil {
		log.Panicf("%s: %s", msg, err)
	}
}

func main() {
	var conn *amqp.Connection
	var err error
	err = godotenv.Load(".env")
	failOnError(err, "Failed to read .env file")
	conn, err = amqp.Dial("amqp://" + os.Getenv("RABBITMQ_USERNAME") + ":" + os.Getenv("RABBITMQ_PASS") + "@rabbitmq:5672/")

	failOnError(err, "Failed to connect to RabbitMQ")
	defer conn.Close()

	redis_client := redis.NewClient(&redis.Options{
		Addr:     "redis-payment:6379",
		Password: "",
		DB:       0,
	})

	_, err = redis_client.Ping(context.Background()).Result()
	failOnError(err, "Failed to connect to Redis")

	log.Println("Connected to Redis")

	ch, err := conn.Channel()
	failOnError(err, "Failed to open a channel")
	defer ch.Close()

	// consumer might exist before the producer, make sure queue exists before consuming msg
	err = ch.ExchangeDeclare("order", "direct", true, false, false, false, nil)
	failOnError(err, "Failed to declare an exchange")

	err = ch.ExchangeDeclare("payment", "direct", true, false, false, false, nil)
	failOnError(err, "Failed to declare an exchange")

	err = ch.ExchangeDeclare("paid", "direct", true, false, false, false, nil)
	failOnError(err, "failed to declare paid exchange")

	q, err := ch.QueueDeclare("", false, false, true, false, nil)
	failOnError(err, "Failed to declare a queue")

	payment_queue, err := ch.QueueDeclare("paym_recv", false, false, true, false, nil)
	failOnError(err, "Failed to declare a queue")

	log.Printf("Binding queue %s to exchange %s with routing key orderch", q.Name, "logs_direct")
	err = ch.QueueBind(
		q.Name,
		"orderch",
		"order",
		false,
		nil,
	)
	failOnError(err, "Failed to bind a queue")

	log.Printf("Binding queue %s to exchange %s with no route key", q.Name, "payment")
	err = ch.QueueBind(
		payment_queue.Name,
		"",
		"payment",
		false,
		nil,
	)
	failOnError(err, "Failed to bind a queue")

	msgs, err := ch.Consume(q.Name, "", false, false, false, false, nil)
	failOnError(err, "Failed to register a consumer")

	payment_msgs, err := ch.Consume(payment_queue.Name, "", false, false, false, false, nil)
	failOnError(err, "Failed to register a consumer")

	var forever chan struct{}

	go func() { // payment success from stripe (fetch current order from redis)
		for d := range payment_msgs {
			id := string(d.Body[:])
			text, err := redis_client.Get(context.Background(), id).Result()
			if err != nil {
				log.Fatalf("id %s err in redis get, err: %s", id, err)
			}
			var order *pb.Order
			err = json.Unmarshal([]byte(text), &order)
			if err != nil {
				log.Print("Received message cannot be parsed by JSON: ", err)
				continue
			}
			order.Status = "Paid"
			d.Ack(false)

			body, err := json.Marshal(order)
			if err != nil {
				failOnError(err, "failed to marshal JSON")
			}
			err = ch.PublishWithContext(context.Background(), "paid", order.GetOrderMachineID(), false, false, amqp.Publishing{
				ContentType: "application/json",
				Body:        body,
			})
			failOnError(err, "failed to publish a message to paid exchange")
			log.Printf("[x] Sent %s \n", body)
		}
	}()

	go func() { // sending to stripe (store current order in redis)
		for d := range msgs {
			var order *pb.Order
			err := json.Unmarshal(d.Body, &order)
			if err != nil {
				log.Print("Received message cannot be parsed by JSON: ", err)
				continue
			}
			err = redis_client.Set(context.Background(), order.ID, d.Body, 0).Err()
			if err != nil {
				log.Print("Cannot set to redis: ", err)
			}
			log.Printf("Received a message: %s", order)
			url, err := stripe.CreatePaymentLink(order)
			if err != nil {
				log.Print("Error from Stripe: ", err)
				continue
			}
			log.Println("Payment URL: ", url)
			log.Printf("Waiting for payment...")
		}
	}()

	log.Printf("[*] Waiting for RPC requests...")
	<-forever
}
