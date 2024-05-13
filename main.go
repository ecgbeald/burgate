package main

import (
	"flag"
	"log"
	"net/http"
	"os"

	pb "github.com/ecgbeald/burgate/proto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func main() {
	status := flag.Bool("local", false, "toggle for local deployment")
	flag.Parse()
	if *status {
		log.Print("Running Locally")
	}

	addr := os.Getenv("ADDR")

	var conn *grpc.ClientConn
	var err error
	if *status {
		conn, err = grpc.Dial(":8889", grpc.WithTransportCredentials(insecure.NewCredentials()))
	} else {
		conn, err = grpc.Dial(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	}
	if err != nil {
		log.Fatal("Fail to dial server", err)
	}
	defer conn.Close()

	orderCli := pb.NewOrderServiceClient(conn)
	menuCli := pb.NewMenuServiceClient(conn)

	mux := http.NewServeMux()
	handler := NewHandler(orderCli, menuCli)
	handler.registerRoutes(mux)

	log.Printf("[SYS] listening on :8888...")

	if err := http.ListenAndServe(":8888", mux); err != nil {
		log.Fatal("failed to start http server")
	}
}
