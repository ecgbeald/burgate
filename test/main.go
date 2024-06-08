package main

import (
	"context"
	"encoding/json"
	"fmt"

	pb "github.com/ecgbeald/burgate/proto"
	"github.com/redis/go-redis/v9"
)

func main() {
	client := redis.NewClient(&redis.Options{
		Addr:     "localhost:6379",
		Password: "",
		DB:       0,
	})
	pong, err := client.Ping(context.Background()).Result()
	fmt.Println(pong, err)

	var idk []*pb.Item
	idk = append(idk, &pb.Item{
		ID:       "123",
		Name:     "456",
		Quantity: 999,
		PriceID:  "999",
	})
	idk = append(idk, &pb.Item{
		ID:       "334",
		Name:     "44",
		Quantity: 999,
		PriceID:  "23",
	})

	item := &pb.Order{
		ID:             "123",
		CustomerID:     "234",
		Status:         "dead",
		Items:          idk,
		OrderMachineID: "0",
	}

	json, err := json.Marshal(item)
	if err != nil {
		fmt.Println(err)
	}

	ctx := context.Background()

	err = client.Set(ctx, "123", json, 0).Err()
	if err != nil {
		fmt.Println(err)
	}

	val, err := client.Get(ctx, "123").Result()
	if err != nil {
		fmt.Println(err)
	}
	fmt.Println(val)

}
