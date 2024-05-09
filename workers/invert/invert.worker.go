package main

import (
	"bufio"
	"bytes"
	"fmt"
	"image"
	"image/jpeg"
	"io"
	"log"

	"github.com/disintegration/imaging"
)

func (i *Image) processImageBuffer(inch *chan ImageChunk, outch *chan ImageChunk) error {
	var (
		imgBuffer = new(bytes.Buffer)
		rImgBuf   = new(bytes.Buffer)
		eof       = false
	)

	for {
		select {
		case img_chunk := <-*inch:
			if img_chunk.completed {
				eof = true
			} else {
				imgBuffer.Write(img_chunk.data)
			}
		default:
			if eof {
				invertedImg, err := i.invertImageColors(imgBuffer)
				if err != nil {
					fmt.Printf("Error decoding: %s\n", err.Error())
					continue
				}
				err = jpeg.Encode(rImgBuf, invertedImg, nil)
				if err != nil {
					log.Fatalf("Error: %v", err)
					return err
				}

				go i.pipeResult(rImgBuf, outch)

				imgBuffer.Reset()
				eof = false
			}
		}
	}
}

func (i *Image) invertImageColors(r io.Reader) (image.Image, error) {
	img, err := jpeg.Decode(r)
	if err != nil {
		return nil, err
	}
	return imaging.Invert(img), nil
}

func (i *Image) pipeResult(b *bytes.Buffer, c *chan ImageChunk) {
	var (
		reader    = bufio.NewReader(b)
		chunkSize = 64
	)

	for {
		chunk := make([]byte, chunkSize)
		n, err := reader.Read(chunk)
		if err != nil {
			if err == io.EOF {
				*c <- NewImageChunk(nil, 0, 0, true)
				b.Reset()
				break
			} else {
				fmt.Println("Error reading chunk:", err)
				break
			}
		}
		*c <- NewImageChunk(chunk[:n], 100, 100, false)
	}
}
