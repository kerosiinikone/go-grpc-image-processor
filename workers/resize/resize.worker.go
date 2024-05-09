package main

import (
	"bytes"
	"fmt"
	"image"
	"image/jpeg"
	"io"
	"log"

	u "github.com/kerosiinikone/go-docker-grpc/workers"
	"github.com/nfnt/resize"
)

const processor = resize.Lanczos3

// processImage reads the image chunks from the channel and buffers them to be eventually
// processed by a utility function
func processImage(inch *chan u.ImageChunk, outch *chan u.ImageChunk) error {
	var (
		imgBuffer = new(bytes.Buffer)
		rImgBuf   = new(bytes.Buffer)
		eof       = false
		i         = u.Image{}
	)

	for {
		select {
		case img_chunk := <-*inch:
			if img_chunk.Completed {
				eof = true
			} else {
				imgBuffer.Write(img_chunk.Data)
			}
		default:
			if eof {
				resizedImg, err := resizeImage(imgBuffer, 100, 100) // hardcoded for now
				if err != nil {
					fmt.Printf("Error decoding: %s\n", err.Error())
					continue
				}
				err = jpeg.Encode(rImgBuf, resizedImg, nil)
				if err != nil {
					log.Fatalf("Error: %v", err)
					return err
				}
				// Successful Image Resizing
				go i.PipeResult(rImgBuf, outch)

				imgBuffer.Reset()
				eof = false
			}
		}
	}
}

// resizeImage is a utility function that resizes an image from a reader and returns
// an image.Image type
func resizeImage(r io.Reader, newHeight int32, newWidth int32) (image.Image, error) {
	img, err := jpeg.Decode(r)
	if err != nil {
		return nil, err
	}
	return resize.Resize(uint(newHeight), uint(newWidth), img, processor), nil
}
