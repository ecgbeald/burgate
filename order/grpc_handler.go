package main

import (
	"context"
	"encoding/json"
	"log"
	"os"
	"strconv"
	"time"

	pb "github.com/ecgbeald/burgate/proto"
	"github.com/ecgbeald/burgate/stripe"
	"github.com/google/uuid"
	amqp "github.com/rabbitmq/amqp091-go"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/emptypb"
)

type grpc_order_handler struct {
	store   OrderStore
	service OrderService
	ch      *amqp.Channel
	pb.UnimplementedOrderServiceServer
}

type grpc_menu_handler struct {
	store OrderStore
	pb.UnimplementedMenuServiceServer
}

func NewGRPCHandler(grpcServer *grpc.Server, store OrderStore, service OrderService, ch *amqp.Channel) {
	orderHandler := &grpc_order_handler{store: store, service: service, ch: ch}
	menuHandler := &grpc_menu_handler{store: store}
	pb.RegisterOrderServiceServer(grpcServer, orderHandler)
	pb.RegisterMenuServiceServer(grpcServer, menuHandler)
}

func (gh *grpc_menu_handler) CreateMenuEntry(ctx context.Context, entries *pb.MenuEntries) (*pb.DbResponse, error) {
	log.Print("Received: ", entries.Entries)
	resp := "ok"
	var menu []menu_entry_db
	for _, entry := range entries.Entries {
		price, err := strconv.Atoi(entry.Price)
		if err != nil {
			continue // bad format, ignore
		}
		priceID, err := stripe.CreateStripeProduct(entry.EntryName, int64(price))
		if err != nil {
			log.Println("stripe error: ", err)
		}
		menu = append(menu, menu_entry_db{Id: entry.ID, Name: entry.EntryName, Price: entry.Price, PriceID: priceID})
	}
	if err := gh.store.Create(ctx, &menu); err != nil {
		return &pb.DbResponse{
			Error:    true,
			ErrorMsg: &resp,
		}, nil
	}
	return &pb.DbResponse{
		Error:    false,
		ErrorMsg: &resp,
	}, nil
}

func (gh *grpc_order_handler) CreateOrder(ctx context.Context, request *pb.CreateOrderRequest) (*pb.Order, error) {
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

func (gh *grpc_order_handler) ReceiveCookedOrder(ctx context.Context, order *pb.Order) (*emptypb.Empty, error) {
	log.Print("Received: ", order)
	// TODO
	return &emptypb.Empty{}, nil
}

func validateQuery(ctx context.Context, store OrderStore, item *pb.ItemsWithQuantity) (*pb.Item, error) {
	result, err := store.Query(ctx, item.ID)
	if err != nil {
		return nil, err
	}
	return &pb.Item{
		ID:       item.ID,
		Name:     result.Name,
		Quantity: item.Quantity,
		PriceID:  result.PriceID, // TODO
	}, nil
}

func convertItem(ctx context.Context, request *pb.CreateOrderRequest, store OrderStore) *pb.Order {
	uuid := uuid.New()
	var items []*pb.Item
	custID := request.CustomerID
	for _, item := range request.Items {
		res, err := validateQuery(ctx, store, item)
		if err != nil {
			log.Print("Failed to fetch from database:", err)
			continue
		}
		items = append(items, res)
	}
	return &pb.Order{
		ID:             uuid.String(),
		CustomerID:     custID,
		Status:         "Ordered",
		Items:          items,
		OrderMachineID: os.Getenv("ID"),
	}
}
