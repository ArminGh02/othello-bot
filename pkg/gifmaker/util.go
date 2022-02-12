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

func cloneImage(src *image.Paletted) *image.Paletted {
	clone := *src
	clone.Pix = make([]uint8, len(src.Pix))
	copy(clone.Pix, src.Pix)
	return &clone
}
