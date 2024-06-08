package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"

	"github.com/joho/godotenv"
	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/stripe/stripe-go/v78"
	"github.com/stripe/stripe-go/v78/webhook"
)

// The library needs to be configured with your account's secret key.
// Ensure the key is kept out of any version control system you might be using.

func main() {
	err := godotenv.Load(".env")
	if err != nil {
		log.Fatal("Error loading .env")
	}
	stripe.Key = os.Getenv("STRIPE_API_KEY")
	conn, err := amqp.Dial("amqp://guest:guest@localhost:5672/")
	if err != nil {
		log.Panicf("%s: %s", "Failed to connect to RabbitMQ", err)
	}
	defer conn.Close()

	ch, err := conn.Channel()
	if err != nil {
		log.Panicf("%s: %s", "Failed to open a channel", err)
	}
	defer ch.Close()

	log.Printf("Connected to RabbitMQ")

	err = ch.ExchangeDeclare("payment", "direct", true, false, false, false, nil)
	if err != nil {
		log.Panicf("%s: %s", "Failed to declare payment exchange", err)
	}

	webhookHandler := func(w http.ResponseWriter, req *http.Request) {
		stripe.Key = os.Getenv("STRIPE_API_KEY")

		const MaxBodyBytes = int64(65536)
		req.Body = http.MaxBytesReader(w, req.Body, MaxBodyBytes)
		payload, err := io.ReadAll(req.Body)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error reading request body: %v\n", err)
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}

		// This is your Stripe CLI webhook secret for testing your endpoint locally.
		endpointSecret := "whsec_f866bf06ac27f0dc136228f2c6f492e0633cf32f93523b9e7f3b91c01481a3c4"
		// Pass the request body and Stripe-Signature header to ConstructEvent, along
		// with the webhook signing key.
		event, err := webhook.ConstructEvent(payload, req.Header.Get("Stripe-Signature"),
			endpointSecret)

		if err != nil {
			fmt.Fprintf(os.Stderr, "Error verifying webhook signature: %v\n", err)
			w.WriteHeader(http.StatusBadRequest) // Return a 400 error on a bad signature
			return
		}

		// Unmarshal the event data into an appropriate struct depending on its Type
		switch event.Type {
		// case "payment_intent.succeeded":
		// Then define and call a function to handle the event payment_intent.succeeded
		// ... handle other event types
		case "checkout.session.completed":
			log.Println("Checkout Complete")
			var session stripe.CheckoutSession
			err := json.Unmarshal(event.Data.Raw, &session)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error parsing webhook JSON: %v\n", err)
				w.WriteHeader(http.StatusBadRequest)
				return
			}
			order_id, ok := session.Metadata["order_id"]
			if !ok {
				fmt.Fprintf(os.Stderr, "Error fetching predefined order_id: %v\n", err)
				w.WriteHeader(http.StatusBadRequest)
				return
			}

			err = ch.PublishWithContext(context.Background(), "payment", "", false, false, amqp.Publishing{
				ContentType: "application/json",
				Body:        []byte(order_id),
			})
			if err != nil {
				log.Panicf("%s: %s", "failed to publish a message to payment exchange", err)
			}

		default:
			fmt.Fprintf(os.Stderr, "Unhandled event type: %s\n", event.Type)
		}

		w.WriteHeader(http.StatusOK)
	}

	http.HandleFunc("/webhook", webhookHandler)
	addr := "localhost:4242"
	log.Printf("Listening on %s", addr)
	log.Fatal(http.ListenAndServe(addr, nil))

}
