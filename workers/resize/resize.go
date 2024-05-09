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
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
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
	inch   chan ImageChunk
	outch  chan ImageChunk
	rechan chan ImageChunk
	cfg    c.Config

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

	go handleProxyImageService(&svc.cfg, &svc.outch, &svc.rechan)

	// Send final to client -> listen to the following service
	wg.Add(1)
	go func() {
		for final := range svc.rechan {
			if final.completed {
				wg.Done()
				return
			} else {
				resp := img_grpc.Image{
					ImageData:   final.data,
					ImageWidth:  final.width,
					ImageHeight: final.height,
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
				svc.inch <- NewImageChunk(nil, 0, 0, true)
				wg.Wait()
				return nil
			}
			if err != nil {
				// Retry
				continue
			}
			// Pipe to worker
			svc.inch <- NewImageChunk(image_data_req.ImageData, image_data_req.ImageHeight, image_data_req.ImageWidth, false)
		}
	}
}

func startServerAndListen(cfg *c.Config, in *chan ImageChunk, out *chan ImageChunk, re *chan ImageChunk) {
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

func handleProxyImageService(cfg *c.Config, outch *chan ImageChunk, rechan *chan ImageChunk) {
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
		fmt.Println("Client sender started")
		for img_chunk := range *outch {
			var req img_grpc.Image

			if img_chunk.completed {
				req = img_grpc.Image{
					ImageData: nil,
				}
				if err := stream.Send(&req); err != nil {
					log.Fatalf("can not send %v", err)
				}
				fmt.Println("Client sender returned")
				return
			} else {
				req = img_grpc.Image{
					ImageData:   img_chunk.data,
					ImageHeight: img_chunk.height,
					ImageWidth:  img_chunk.width,
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
		fmt.Println("Client receiver started")
		for {
			resp, err := stream.Recv()
			if err != nil {
				continue
			}
			if resp.ImageData == nil {
				*rechan <- NewImageChunk(nil, 0, 0, true) // Signal Completion
				wg.Done()
				fmt.Println("Client receiver returned")
				return
			}
			*rechan <- NewImageChunk(resp.ImageData, resp.ImageHeight, resp.ImageWidth, false)
		}
	}()
	wg.Wait()
}

func main() {
	var (
		cfg    = c.Load()
		inch   = make(chan ImageChunk) // Unprocessed
		outch  = make(chan ImageChunk) // Processed
		rechan = make(chan ImageChunk) // Final
		i      = Image{}
	)

	// Starts the workers
	go i.processImageBuffer(&inch, &outch)

	// Starts the API (blocking)
	startServerAndListen(cfg, &inch, &outch, &rechan)
}
