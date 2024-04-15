package main

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"sync"

	c "github.com/kerosiinikone/go-docker-grpc/config"
	img_grpc "github.com/kerosiinikone/go-docker-grpc/grpc"
	grpc "google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// TODO: Ability to load all image types

const input_img = "input.jpg"

type Image struct {
	height 	int32
	width 	int32
	chunk 	[]byte
}

func (i *Image) loadImageChunks(outch *chan Image) {
	var (
		chunkSize = 64
	)
	
	file, err := os.Open(input_img)
	
	if err != nil {
		log.Fatalf("Open -> %v", err)
	}

	defer file.Close()

	reader := bufio.NewReader(file)

	// TODO: Get image config from chunks, etc

	for {
		chunk := make([]byte, chunkSize)
		_, err := reader.Read(chunk)
		if err == io.EOF {
			close(*outch)
			fmt.Printf("End of data")
			return
		}
		if err != nil {
			log.Fatalf("Read -> %v", err)
		}

		fmt.Printf("Read image: %v\n", chunk)
		// NewImage, etc
		*outch <- Image{
			height: 0,
			width: 	0,
			chunk: 	chunk,
		}
	}
}

func main() {
	var (
		cfg = 	c.Load()
		outch = make(chan Image)
		wg sync.WaitGroup
		i Image
	)

	conn, err := grpc.Dial(
		fmt.Sprintf("%s:%d1", cfg.Server.Addr, cfg.Server.Port), 
		grpc.WithTransportCredentials(insecure.NewCredentials()))

	if err != nil {
		log.Fatalf("can not connect with server %v", err)
	}

	client := img_grpc.NewImageServiceClient(conn)

	// Open stream
	stream, err := client.TransferImageBytes(context.Background())

	if err != nil {
		log.Fatalf("open stream error %v", err)
	}

	// Load the image concurrently
	go i.loadImageChunks(&outch)

	// Propagation
	// ctx := stream.Context()

	go func() {
		for img_chunk := range outch {
			req := img_grpc.Image{
				ImageData: img_chunk.chunk,
				ImageHeight: img_chunk.height,
				ImageWidth: img_chunk.width,
			}
			if err := stream.Send(&req); err != nil {
				log.Fatalf("can not send %v", err)
			}
		}
		if err := stream.CloseSend(); err != nil {
			log.Fatalf("%v", err)
		}
	}()

	wg.Add(1)
	go func() {
		for {
			resp, err := stream.Recv()
			if err == io.EOF {
				// Return or continue based on whether you want
				// to continue streaming from the channel or restart 
				wg.Done()
				return
			}
			if err != nil {
				log.Fatalf("can not receive %v", err)
			}

			img_data := resp.ImageData
			log.Printf("new chunk %v received", img_data)
		}
	}()

	wg.Wait()
}
