package main

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log"
	"sync"

	c "github.com/kerosiinikone/go-docker-grpc/config"
	img_grpc "github.com/kerosiinikone/go-docker-grpc/grpc"
	grpc "google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// func connectToServer()  {}
// func handleConnection() {}

func streamImage(outch *chan Image, stream img_grpc.ImageService_TransferImageBytesClient) error {
	for img_chunk := range *outch {
		req := img_grpc.Image{
			ImageData:   img_chunk.chunk,
			ImageHeight: img_chunk.height,
			ImageWidth:  img_chunk.width,
		}
		if err := stream.Send(&req); err != nil {
			return err
		}
	}
	if err := stream.CloseSend(); err != nil {
		return err
	}

	return nil
}

func receiveImage(stream img_grpc.ImageService_TransferImageBytesClient) error {
	finalImageBytes := new(bytes.Buffer)

	for {
		resp, err := stream.Recv()

		// Signal completion
		if err == io.EOF {
			if err := saveImage(finalImageBytes); err != nil {
				return err
			}
			return nil
		}
		if err != nil {
			continue
		}
		img_data := resp.ImageData
		finalImageBytes.Write(img_data)
	}
}

func main() {
	var (
		cfg   = c.Load()
		outch = make(chan Image)
		wg    sync.WaitGroup
	)

	conn, err := grpc.Dial(
		fmt.Sprintf("%s:%d1", cfg.Server.Addr, cfg.Server.Port),
		grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("can not connect with server %v", err)
	}
	client := img_grpc.NewImageServiceClient(conn)

	defer conn.Close()

	// Open stream
	stream, err := client.TransferImageBytes(context.Background())
	if err != nil {
		log.Fatalf("open stream error %v", err)
	}

	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := loadImage(&outch); err != nil {
			log.Fatalf("Error loading image: %v", err)
		}
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := streamImage(&outch, stream); err != nil {
			log.Fatalf("Error streaming image: %v", err)
		}
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := receiveImage(stream); err != nil {
			log.Fatalf("Error saving image: %v", err)
		}
	}()

	wg.Wait()
}
