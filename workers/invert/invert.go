package main

import (
	"fmt"
	"log"
	"net"
	"sync"

	c "github.com/kerosiinikone/go-docker-grpc/config"
	img_grpc "github.com/kerosiinikone/go-docker-grpc/grpc"
	"google.golang.org/grpc"
)

// Placehoder for a Processor interface (later)
type Image struct{}

// ImageChunk holds data that is relevant when chunks of an image are
// passed around internally
type ImageChunk struct {
	data      []byte
	height    int32
	width     int32
	completed bool
}

// apiService is a gRPC placeholder that is augmented
// with logic for concurrent processing
type apiService struct {
	inch  chan ImageChunk
	outch chan ImageChunk

	img_grpc.UnsafeImageServiceServer
}

// Creates a new ImageChunk with an image chunk and image dimensions
func NewImageChunk(d []byte, h int32, w int32, completed bool) ImageChunk {
	return ImageChunk{
		data:      d,
		height:    h,
		width:     w,
		completed: completed,
	}
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
			if processed.completed {
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
					ImageData:   processed.data,
					ImageHeight: processed.height,
					ImageWidth:  processed.width,
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
				svc.inch <- NewImageChunk(nil, 0, 0, true)
				wg.Wait()
				return nil
			}

			// Pipe the data to a worker
			svc.inch <- NewImageChunk(image_data_req.ImageData, image_data_req.ImageHeight, image_data_req.ImageWidth, false)
		}
	}
}

// Listens for TCP connections for a port and an address
// that are set in the config
func startServerAndListen(cfg *c.Config, in *chan ImageChunk, out *chan ImageChunk) {
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
		cfg   = c.Load()              // Holds the server addresses
		inch  = make(chan ImageChunk) // Unprocessed
		outch = make(chan ImageChunk) // Processed
		i     = Image{}
	)

	// Starts the worker
	go func() {
		if err := i.processImage(&inch, &outch); err != nil {
			log.Fatalf("Error processing image: %v", err)
		}
	}()

	// Starts the API (blocking)
	startServerAndListen(cfg, &inch, &outch)
}
