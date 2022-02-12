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

func convertImageToPaletted(img image.Image) *image.Paletted {
	opts := gif.Options{
		NumColors: 256,
		Drawer:    draw.FloydSteinberg,
	}

	res := image.NewPaletted(img.Bounds(), palette.Plan9[:opts.NumColors])
	opts.Drawer.Draw(res, img.Bounds(), img, image.Point{})
	return res
}

func cloneImage(src *image.Paletted) *image.Paletted {
	clone := *src
	clone.Pix = make([]uint8, len(src.Pix))
	copy(clone.Pix, src.Pix)
	return &clone
}
