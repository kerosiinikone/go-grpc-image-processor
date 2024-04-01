package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"time"

	"github.com/kerosiinikone/go-docker-grpc/config"
	gen "github.com/kerosiinikone/go-docker-grpc/grpc"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func runGetUser(client gen.UserServiceClient) {
	// Propagation with context.WithTimeout
	ctx, cancel := context.WithTimeout(context.Background(), time.Second * 5)
	defer cancel()

	fmt.Println("New request to GetUser")
	user, err := client.GetUser(ctx, &gen.Identifier{
		Id: 1,
	})
	if err != nil {
		log.Fatalf("Req failed: %v", err)
	}
	log.Println(user)
}

func runGetUserList(client gen.UserServiceClient) {
	// Propagation with context.WithTimeout
	ctx, cancel := context.WithTimeout(context.Background(), time.Second * 5)
	defer cancel()
	fmt.Println("New request to GetUser")
	stream, err := client.GetUserList(ctx, &gen.EmptyParams{})
	if err != nil {
		log.Fatalf("Req failed: %v", err)
	}
	for {
		in, err := stream.Recv()

		if err == io.EOF {
			break
		}
		if err != nil {
			log.Fatalf("%v", err)
		}

		fmt.Printf("%v", in)
	}

}

func main() {
	cfg := config.New()

	var (
		opts = []grpc.DialOption{
			grpc.WithTransportCredentials(insecure.NewCredentials()),
		}
		serverAdrr = fmt.Sprintf("%s:%d", cfg.Addr, cfg.Port)
	)
	conn, err := grpc.Dial(serverAdrr, opts...)
	if err != nil {
		log.Fatalf("%v", err)
	}

	client := gen.NewUserServiceClient(conn)

	// Call GetUser
	runGetUser(client)
	// Call Stream:GetUserList
	runGetUserList(client)
}

// Bidirectional
