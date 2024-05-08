package main

import (
	"context"
	"encoding/json"
	"log"
	"time"

	pb "github.com/ecgbeald/burgate/proto"
	"github.com/google/uuid"
	amqp "github.com/rabbitmq/amqp091-go"
	"google.golang.org/grpc"
)

type menu_entry struct {
	name     string
	price_id string
}

var menu = map[string]menu_entry{
	"1": {"nuggies", "1"},
	"2": {"oolong", "2"},
}

type grpc_handler struct {
	service OrderService
	ch      *amqp.Channel
	pb.UnimplementedOrderServiceServer
}

func NewGRPCHandler(grpcServer *grpc.Server, service OrderService, ch *amqp.Channel) {
	handler := &grpc_handler{service: service, ch: ch}
	pb.RegisterOrderServiceServer(grpcServer, handler)
}

func (gh *grpc_handler) CreateOrder(ctx context.Context, request *pb.CreateOrderRequest) (*pb.Order, error) {
	log.Print("New order received from cust id: ", request.CustomerID)

	o := ConvertItem(request)

	body, err := json.Marshal(o)
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	err = gh.ch.PublishWithContext(ctx,
		"order",
		"orderch",
		false,
		false,
		amqp.Publishing{
			// DeliveryMode: amqp.Persistent,
			ContentType: "text/plain",
			Body:        body,
		})
	failOnError(err, "Failed to publish a msg")
	log.Printf(" [x] Sent %s \n", body)

	return o, nil
}

func ConvertItem(request *pb.CreateOrderRequest) *pb.Order {
	uuid := uuid.New()
	items := make([]*pb.Item, len(request.Items))
	custID := request.CustomerID
	for i, item := range request.Items {
		converted := menu[item.ID]
		items[i] = &pb.Item{
			ID:       item.ID,
			Name:     converted.name,
			Quantity: item.Quantity,
			PriceID:  converted.price_id,
		}
	}
	return &pb.Order{
		ID:         uuid.String(),
		CustomerID: custID,
		Status:     "Ordered",
		Items:      items,
	}
}
