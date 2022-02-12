package gifmaker

import (
	"image"
	"image/color/palette"
	"image/draw"
	"image/gif"
	"image/png"
	"log"
	"os"
)

func readPNG(filename string) image.Image {
	f, err := os.Open(filename)
	if err != nil {
		log.Panicln(err)
	}

	img, err := png.Decode(f)
	if err != nil {
		log.Panicln(err)
	}
	return img
}

func cloneImage(src *image.Paletted) *image.Paletted {
	clone := *src
	clone.Pix = make([]uint8, len(src.Pix))
	copy(clone.Pix, src.Pix)
	return &clone
}
