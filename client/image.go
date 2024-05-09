package main

import (
	"bufio"
	"image/jpeg"
	"io"
	"os"
)

const input_img = "input.jpg"

type Image struct {
	height int32
	width  int32
	chunk  []byte
}

func loadImage(outch *chan Image) error {
	var (
		chunkSize = 64
	)

	file, err := os.Open(input_img)
	if err != nil {
		return err
	}
	defer file.Close()

	reader := bufio.NewReader(file)

	for {
		chunk := make([]byte, chunkSize)
		_, err := reader.Read(chunk)
		if err == io.EOF {
			close(*outch)
			return nil
		}
		if err != nil {
			return err
		}

		// Image dimensions set as 100 (default)
		*outch <- Image{
			height: 100,
			width:  100,
			chunk:  chunk,
		}
	}
}

func saveImage(r io.Reader) error {
	img, err := jpeg.Decode(r)
	if err != nil {
		return err
	}
	f, err := os.Create("output.jpg")
	if err != nil {
		return err
	}
	defer f.Close()
	if err = jpeg.Encode(f, img, nil); err != nil {
		return err
	}

	return nil
}
