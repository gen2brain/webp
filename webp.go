// Package webp implements an WEBP image decoder based on libwebp compiled to WASM.
package webp

import (
	"errors"
	"image"
	"io"
)

// Errors .
var (
	ErrMemRead  = errors.New("webp: mem read failed")
	ErrMemWrite = errors.New("webp: mem write failed")
	ErrDecode   = errors.New("webp: decode failed")
)

// WEBP represents the possibly multiple images stored in a WEBP file.
type WEBP struct {
	// Decoded images.
	Image []*image.NRGBA
	// Delay times, one per frame, in milliseconds.
	Delay []int
}

// Decode reads a AVIF image from r and returns it as an image.Image.
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

// DecodeConfig returns the color model and dimensions of a AVIF image without decoding the entire image.
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

// DecodeAll reads a WebP image from r and returns the sequential frames and timing information.
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

func init() {
	image.RegisterFormat("webp", "RIFF????WEBPVP8", Decode, DecodeConfig)
}
