package image

import (
	"bytes"
	"errors"
	"image"
	"image/gif"
	"image/jpeg"
	"image/png"
	"io"
	"math"

	"golang.org/x/image/draw"
)

var NoNeed = errors.New("no need to reduce")

type resizeOptions struct {
	limitEdge   int
	jpegQuality int
}

func resizedRect(r image.Rectangle, o *resizeOptions) image.Rectangle {
	if r.Max.X <= o.limitEdge && r.Max.Y <= o.limitEdge {
		return image.ZR
	} else if r.Max.Y >= r.Max.X {
		t := float64((r.Max.X * o.limitEdge))
		return image.Rect(0, 0, int(math.Round(t/float64(r.Max.Y))), o.limitEdge)
	} else {
		t := float64((r.Max.Y * o.limitEdge))
		return image.Rect(0, 0, o.limitEdge, int(math.Round(t/float64(r.Max.X))))
	}
}

type Option func(*resizeOptions)

func SetLimitEdge(l int) Option {
	return func(o *resizeOptions) {
		o.limitEdge = l
	}
}

func Resize(r io.Reader, opts ...Option) (*bytes.Reader, image.Rectangle, error) {
	options := &resizeOptions{
		limitEdge:   960,
		jpegQuality: 100,
	}
	for _, f := range opts {
		f(options)
	}

	img, fmt, err := image.Decode(r)
	if err != nil {
		return nil, image.ZR, err
	}

	newRect := resizedRect(img.Bounds(), options)
	if newRect == image.ZR {
		return nil, image.ZR, NoNeed
	}

	newImg := image.NewRGBA(newRect)
	draw.CatmullRom.Scale(newImg, newImg.Bounds(), img, img.Bounds(), draw.Over, nil)

	var buf bytes.Buffer
	switch fmt {
	case "jpeg":
		if err := jpeg.Encode(&buf, newImg, &jpeg.Options{Quality: options.jpegQuality}); err != nil {
			return nil, image.ZR, err
		}
	case "png":
		if err := png.Encode(&buf, newImg); err != nil {
			return nil, image.ZR, err
		}
	case "gif":
		if err := gif.Encode(&buf, newImg, nil); err != nil {
			return nil, image.ZR, err
		}
	default:
		return nil, image.ZR, errors.New("unsupported format")
	}

	return bytes.NewReader(buf.Bytes()), newImg.Bounds(), err
}
