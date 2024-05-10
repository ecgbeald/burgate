package main

import (
	"flag"
	"log"
	"net/http"

	pb "github.com/ecgbeald/burgate/proto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func main() {
	addr := flag.String("addr", "order:8889", "addr to connect")
	conn, err := grpc.Dial(*addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatal("Fail to dial server", err)
	}
	defer conn.Close()

	log.Println("dialing orders service at :8889")

	cli := pb.NewOrderServiceClient(conn)

	mux := http.NewServeMux()
	handler := NewHandler(cli)
	handler.registerRoutes(mux)

	log.Printf("[SYS] listening on :8888...")

	if err := http.ListenAndServe(":8888", mux); err != nil {
		log.Fatal("failed to start http server")
	}
}
