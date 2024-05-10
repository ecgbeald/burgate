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

type grpc_handler struct {
	store   OrderStore
	service OrderService
	ch      *amqp.Channel
	pb.UnimplementedOrderServiceServer
}

func NewGRPCHandler(grpcServer *grpc.Server, store OrderStore, service OrderService, ch *amqp.Channel) {
	handler := &grpc_handler{store: store, service: service, ch: ch}
	pb.RegisterOrderServiceServer(grpcServer, handler)
}

func (gh *grpc_handler) CreateOrder(ctx context.Context, request *pb.CreateOrderRequest) (*pb.Order, error) {
	log.Print("New order received from cust id: ", request.CustomerID)

	o := convertItem(ctx, request, gh.store)

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

func validateQuery(ctx context.Context, store OrderStore, item *pb.ItemsWithQuantity) (*pb.Item, error) {
	results, err := store.Query(ctx, item.ID)
	if err != nil {
		return nil, err
	}
	mostRecent := (*results)[0]
	return &pb.Item{
		ID:       item.ID,
		Name:     mostRecent.Name,
		Quantity: item.Quantity,
		PriceID:  mostRecent.Price_id,
	}, nil
}

func convertItem(ctx context.Context, request *pb.CreateOrderRequest, store OrderStore) *pb.Order {
	uuid := uuid.New()
	items := make([]*pb.Item, len(request.Items))
	custID := request.CustomerID
	for i, item := range request.Items {
		res, err := validateQuery(ctx, store, item)
		if err != nil {
			log.Fatal("Failed to fetch from database")
		}
		items[i] = res
	}
	return &pb.Order{
		ID:         uuid.String(),
		CustomerID: custID,
		Status:     "Ordered",
		Items:      items,
	}
}
