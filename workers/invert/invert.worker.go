package main

import (
	"bytes"
	"fmt"
	"image"
	"image/jpeg"
	"io"
	"log"

	"github.com/disintegration/imaging"
	u "github.com/kerosiinikone/go-docker-grpc/workers"
)

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
				invertedImg, err := invertImageColors(imgBuffer)
				if err != nil {
					fmt.Printf("Error decoding: %s\n", err.Error())
					continue
				}
				err = jpeg.Encode(rImgBuf, invertedImg, nil)
				if err != nil {
					log.Fatalf("Error: %v", err)
					return err
				}

				go i.PipeResult(rImgBuf, outch)

				imgBuffer.Reset()
				eof = false
			}
		}
	}
}

// invertImageColors is a utility function that inverts the colors of an image.Image from
// an io.Reader type
func invertImageColors(r io.Reader) (image.Image, error) {
	img, err := jpeg.Decode(r)
	if err != nil {
		return nil, err
	}
	return imaging.Invert(img), nil
}
