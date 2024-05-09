package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"net"
	"sync"

	c "github.com/kerosiinikone/go-docker-grpc/config"
	img_grpc "github.com/kerosiinikone/go-docker-grpc/grpc"
	u "github.com/kerosiinikone/go-docker-grpc/workers"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// apiService is a gRPC placeholder that is augmented
// with logic for concurrent processing
type apiService struct {
	inch   chan u.ImageChunk
	outch  chan u.ImageChunk
	rechan chan u.ImageChunk
	cfg    c.Config

	img_grpc.UnsafeImageServiceServer
}

// gRPC magic function that is defined in the .proto files
func (svc *apiService) TransferImageBytes(srv img_grpc.ImageService_TransferImageBytesServer) error {
	var (
		ctx = srv.Context()
		wg  sync.WaitGroup
	)

	go handleProxyImageService(&svc.cfg, &svc.outch, &svc.rechan)

	// Send final to client -> listen to the following service
	wg.Add(1)
	go func() {
		for final := range svc.rechan {
			if final.Completed {
				wg.Done()
				return
			} else {
				resp := img_grpc.Image{
					ImageData:   final.Data,
					ImageWidth:  final.Width,
					ImageHeight: final.Height,
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
			if err == io.EOF {
				svc.inch <- u.NewImageChunk(nil, 0, 0, true)
				wg.Wait()
				return nil
			}
			if err != nil {
				// Retry
				continue
			}
			// Pipe to worker
			svc.inch <- u.NewImageChunk(image_data_req.ImageData, image_data_req.ImageHeight, image_data_req.ImageWidth, false)
		}
	}
}

// Listens for TCP connections for a port and an address
// that are set in the config
func startServerAndListen(cfg *c.Config, in *chan u.ImageChunk, out *chan u.ImageChunk, re *chan u.ImageChunk) {
	lis, err := net.Listen("tcp", fmt.Sprintf("%s:%d1", cfg.Server.Addr, cfg.Server.Port))
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}
	s := grpc.NewServer()
	img_grpc.RegisterImageServiceServer(s, &apiService{
		inch:   *in,
		outch:  *out,
		rechan: *re,
		cfg:    *cfg,
	})
	if err := s.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}

// TODO: Error handling in the proxy

// handleProxyImageService forwards the RPC call of a client to the next worker and handles its
// return stream
func handleProxyImageService(cfg *c.Config, outch *chan u.ImageChunk, rechan *chan u.ImageChunk) {
	var (
		wg sync.WaitGroup
	)

	conn, err := grpc.Dial(
		fmt.Sprintf("%s:%d2", cfg.Server.Addr, cfg.Server.Port),
		grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("can not connect with server %v", err)
	}
	client := img_grpc.NewImageServiceClient(conn)

	stream, err := client.TransferImageBytes(context.Background())
	if err != nil {
		log.Fatalf("open stream error %v", err)
	}

	// Send to service 2 -> receive from worker
	go func() {
		for img_chunk := range *outch {
			var req img_grpc.Image

			if img_chunk.Completed {
				req = img_grpc.Image{
					ImageData: nil,
				}
				if err := stream.Send(&req); err != nil {
					log.Fatalf("can not send %v", err)
				}
				return
			} else {
				req = img_grpc.Image{
					ImageData:   img_chunk.Data,
					ImageHeight: img_chunk.Height,
					ImageWidth:  img_chunk.Width,
				}
				if err := stream.Send(&req); err != nil {
					log.Fatalf("can not send %v", err)
				}
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
			if err != nil {
				continue
			}
			if resp.ImageData == nil {
				*rechan <- u.NewImageChunk(nil, 0, 0, true) // Signal Completion
				wg.Done()
				return
			}
			*rechan <- u.NewImageChunk(resp.ImageData, resp.ImageHeight, resp.ImageWidth, false)
		}
	}()
	wg.Wait()
}

func main() {
	var (
		cfg    = c.Load()
		inch   = make(chan u.ImageChunk) // Unprocessed
		outch  = make(chan u.ImageChunk) // Processed
		rechan = make(chan u.ImageChunk) // Final
	)

	// Starts the workers
	go func() {
		if err := processImage(&inch, &outch); err != nil {
			log.Fatalf("Error processing image: %v", err)
		}
	}()

	// Starts the API (blocking)
	startServerAndListen(cfg, &inch, &outch, &rechan)
}
