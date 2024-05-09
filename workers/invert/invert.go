package main

import (
	"fmt"
	"log"
	"net"
	"sync"

	c "github.com/kerosiinikone/go-docker-grpc/config"
	img_grpc "github.com/kerosiinikone/go-docker-grpc/grpc"
	u "github.com/kerosiinikone/go-docker-grpc/workers"
	"google.golang.org/grpc"
)

// apiService is a gRPC placeholder that is augmented
// with logic for concurrent processing
type apiService struct {
	inch  chan u.ImageChunk
	outch chan u.ImageChunk

	img_grpc.UnsafeImageServiceServer
}

// gRPC magic function that is defined in the .proto files
func (svc *apiService) TransferImageBytes(srv img_grpc.ImageService_TransferImageBytesServer) error {
	var (
		ctx = srv.Context()
		wg  sync.WaitGroup
	)

	wg.Add(1)
	go func() {
		for processed := range svc.outch {
			if processed.Completed {
				resp := img_grpc.Image{
					ImageData: nil,
				}
				if err := srv.Send(&resp); err != nil {
					log.Printf("%+v", err)
				}
				wg.Done()
				return
			} else {
				resp := img_grpc.Image{
					ImageData:   processed.Data,
					ImageHeight: processed.Height,
					ImageWidth:  processed.Width,
				}
				if err := srv.Send(&resp); err != nil {
					log.Printf("%+v", err)
				}
			}
		}
	}()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			image_data_req, err := srv.Recv()
			if err != nil {
				// Retry
				continue
			}
			// Signal completion with an empty ImageChunk
			if image_data_req.ImageData == nil {
				svc.inch <- u.NewImageChunk(nil, 0, 0, true)
				wg.Wait()
				return nil
			}

			// Pipe the data to a worker
			svc.inch <- u.NewImageChunk(image_data_req.ImageData, image_data_req.ImageHeight, image_data_req.ImageWidth, false)
		}
	}
}

// Listens for TCP connections for a port and an address
// that are set in the config
func startServerAndListen(cfg *c.Config, in *chan u.ImageChunk, out *chan u.ImageChunk) {
	lis, err := net.Listen("tcp", fmt.Sprintf("%s:%d2", cfg.Server.Addr, cfg.Server.Port))
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}
	s := grpc.NewServer()
	img_grpc.RegisterImageServiceServer(s, &apiService{
		inch:  *in,
		outch: *out,
	})
	if err := s.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}

func main() {
	var (
		cfg   = c.Load()                // Holds the server addresses
		inch  = make(chan u.ImageChunk) // Unprocessed
		outch = make(chan u.ImageChunk) // Processed
	)

	// Starts the worker
	go func() {
		if err := processImage(&inch, &outch); err != nil {
			log.Fatalf("Error processing image: %v", err)
		}
	}()

	// Starts the API (blocking)
	startServerAndListen(cfg, &inch, &outch)
}
