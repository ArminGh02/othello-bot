package gifmaker

import (
	"image"
	"image/draw"
	"log"
)

func cloneImage(src image.Image) draw.Image {
	switch s := src.(type) {
	case *image.Alpha:
		clone := *s
		clone.Pix = clonePix(s.Pix)
		return &clone
	case *image.Alpha16:
		clone := *s
		clone.Pix = clonePix(s.Pix)
		return &clone
	case *image.Gray:
		clone := *s
		clone.Pix = clonePix(s.Pix)
		return &clone
	case *image.Gray16:
		clone := *s
		clone.Pix = clonePix(s.Pix)
		return &clone
	case *image.NRGBA:
		clone := *s
		clone.Pix = clonePix(s.Pix)
		return &clone
	case *image.NRGBA64:
		clone := *s
		clone.Pix = clonePix(s.Pix)
		return &clone
	case *image.RGBA:
		clone := *s
		clone.Pix = clonePix(s.Pix)
		return &clone
	case *image.RGBA64:
		clone := *s
		clone.Pix = clonePix(s.Pix)
		return &clone
	default:
		log.Panicf("gifmaker.cloneImage: unknown image.Image subclass: %T\n", s)
		return nil  // although this statement is unreachable, Go forces me to write it!
	}
}

func clonePix(b []uint8) []uint8 {
	c := make([]uint8, len(b))
	copy(c, b)
	return c
}
