package util

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
)

type Processor interface{}

// Placehoder for a Processor interface (later)
type Image struct{}

// ImageChunk holds data that is relevant when chunks of an image are
// passed around internally
type ImageChunk struct {
	Data      []byte
	Height    int32
	Width     int32
	Completed bool
}

// Creates a new ImageChunk with an image chunk and image dimensions
func NewImageChunk(d []byte, h int32, w int32, completed bool) ImageChunk {
	return ImageChunk{
		Data:      d,
		Height:    h,
		Width:     w,
		Completed: completed,
	}
}

func (i *Image) PipeResult(b *bytes.Buffer, c *chan ImageChunk) {
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
				return
			} else {
				fmt.Println("Error reading chunk:", err)
				return
			}
		}
		*c <- NewImageChunk(chunk[:n], 100, 100, false)
	}
}
