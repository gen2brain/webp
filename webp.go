// Package webp implements an WEBP image decoder based on libwebp compiled to WASM.
package webp

import (
	"errors"
	"image"
	"image/draw"
	"io"
)

// Errors .
var (
	ErrMemRead  = errors.New("webp: mem read failed")
	ErrMemWrite = errors.New("webp: mem write failed")
	ErrDecode   = errors.New("webp: decode failed")
	ErrEncode   = errors.New("webp: encode failed")
)

const (
	maxWebpHeaderSize = 32
)

// WEBP represents the possibly multiple images stored in a WEBP file.
type WEBP struct {
	// Decoded images.
	Image []*image.NYCbCrA
	// Delay times, one per frame, in milliseconds.
	Delay []int
}

// DefaultQuality is the default quality encoding parameter.
const DefaultQuality = 75

// Options are the encoding parameters.
type Options struct {
	// Quality in the range [0,100]. Quality of 100 implies Lossless. Default is 75.
	Quality int
	// Lossless indicates whether to use the lossless compression. Lossless will ignore quality.
	Lossless bool
}

// Decode reads a WEBP image from r and returns it as an image.Image.
func Decode(r io.Reader) (image.Image, error) {
	var err error
	var ret *WEBP

	if dynamic {
		ret, _, err = decodeDynamic(r, false, false)
		if err != nil {
			return nil, err
		}
	} else {
		ret, _, err = decode(r, false, false)
		if err != nil {
			return nil, err
		}
	}

	return ret.Image[0], nil
}

// DecodeConfig returns the color model and dimensions of a WEBP image without decoding the entire image.
func DecodeConfig(r io.Reader) (image.Config, error) {
	var err error
	var cfg image.Config

	if dynamic {
		_, cfg, err = decodeDynamic(r, true, false)
		if err != nil {
			return image.Config{}, err
		}
	} else {
		_, cfg, err = decode(r, true, false)
		if err != nil {
			return image.Config{}, err
		}
	}

	return cfg, nil
}

// DecodeAll reads a WEBP image from r and returns the sequential frames and timing information.
func DecodeAll(r io.Reader) (*WEBP, error) {
	var err error
	var ret *WEBP

	if dynamic {
		ret, _, err = decodeDynamic(r, false, true)
		if err != nil {
			return nil, err
		}
	} else {
		ret, _, err = decode(r, false, true)
		if err != nil {
			return nil, err
		}
	}

	return ret, nil
}

// Encode writes the image m to w with the given options.
func Encode(w io.Writer, m image.Image, o ...Options) error {
	lossless := false
	quality := DefaultQuality

	if o != nil {
		opt := o[0]
		lossless = opt.Lossless
		quality = opt.Quality

		if quality <= 0 {
			quality = DefaultQuality
		} else if quality > 100 {
			quality = 100
		}
	}

	if dynamic {
		err := encodeDynamic(w, m, quality, lossless)
		if err != nil {
			return err
		}
	} else {
		err := encode(w, m, quality, lossless)
		if err != nil {
			return err
		}
	}

	return nil
}

// Dynamic returns error (if there was any) during opening dynamic/shared library.
func Dynamic() error {
	return dynamicErr
}

func imageToNRGBA(src image.Image) *image.NRGBA {
	if dst, ok := src.(*image.NRGBA); ok {
		return dst
	}

	b := src.Bounds()
	dst := image.NewNRGBA(b)
	draw.Draw(dst, dst.Bounds(), src, b.Min, draw.Src)

	return dst
}

func init() {
	image.RegisterFormat("webp", "RIFF????WEBPVP8", Decode, DecodeConfig)
}
