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

type Image struct{}

// Internal Data -> attach metadata to chunks of data later
type ImageChunk struct {
	data      []byte
	height    int32
	width     int32
	completed bool
}

// Augmented gRPC logic for concurrent processing
type apiService struct {
	inch  chan ImageChunk
	outch chan ImageChunk

	img_grpc.UnsafeImageServiceServer
}

func NewImageChunk(d []byte, h int32, w int32, completed bool) ImageChunk {
	return ImageChunk{
		data:      d,
		height:    h,
		width:     w,
		completed: completed,
	}
}

func (svc *apiService) TransferImageBytes(srv img_grpc.ImageService_TransferImageBytesServer) error {
	var (
		ctx = srv.Context()
		wg  sync.WaitGroup
	)

	// Send final to client -> listen to the following service
	wg.Add(1)
	go func() {
		for final := range svc.outch {
			if final.completed {
				wg.Done()
				return
			} else {
				fmt.Printf("Sending final to client from resize: %+v\n", final.data)
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

func startServerAndListen(cfg *c.Config, in *chan ImageChunk, out *chan ImageChunk) {
	lis, err := net.Listen("tcp", fmt.Sprintf("%s:%d1", cfg.Server.Addr, cfg.Server.Port))
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
		for img_chunk := range *outch {
			var req img_grpc.Image

			if img_chunk.completed {
				req = img_grpc.Image{
					ImageData:   nil,
					ImageHeight: 0,
					ImageWidth:  0,
				}
			} else {
				req = img_grpc.Image{
					ImageData:   img_chunk.data,
					ImageHeight: img_chunk.height,
					ImageWidth:  img_chunk.width,
				}
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
			if err != nil {
				log.Fatalf("can not receive %v", err)
			}
			if resp.ImageData == nil {
				*rechan <- NewImageChunk(nil, 0, 0, true) // Signal Completion
			} else {
				*rechan <- NewImageChunk(resp.ImageData, resp.ImageHeight, resp.ImageWidth, false)
			}
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

	// Client - Pipe
	go handleProxyImageService(cfg, &outch, &rechan)
	// Start Workers
	go i.processImageBuffer(&inch, &outch)
	// API
	startServerAndListen(cfg, &inch, &rechan)
}
