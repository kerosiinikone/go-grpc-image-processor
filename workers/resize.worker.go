package main

import (
	"bytes"
	"image"
	"image/jpeg"
	"log"

	"github.com/nfnt/resize"
)

const processor = resize.Lanczos3

func (i *Image) processImageBuffer(inch *chan ImageChunk, outch *chan ImageChunk) error {
	var (
		imgBuffer []byte
		buf = new(bytes.Buffer)
	)

	for {
		select {
		case img_chunk := <- *inch:
			imgBuffer = append(imgBuffer, img_chunk.data...)
		default:
			if len(imgBuffer) == 0 {
				continue
			} else {
				newImg, _, err := image.Decode(bytes.NewReader(imgBuffer))
				if err != nil {
					continue
				}

				// Resize
				resizedImg := i.resize(newImg, 100, 100)

				err = jpeg.Encode(buf, resizedImg, nil)
				if err != nil {
					log.Fatalf("Error: %v", err)
					return err
				}
				end_result := buf.Bytes()

				*outch <- NewImageChunk(end_result, 100, 100)

				// Signal Completion
				*outch <- NewImageChunk(nil, 0, 0)

				imgBuffer = imgBuffer[:0]
				buf.Reset()
			}
		}
		
	}
}

func (i *Image) resize(img image.Image, newHeight int32, newWidth int32) image.Image {
	return resize.Resize(uint(newHeight), uint(newWidth), img, processor)
}