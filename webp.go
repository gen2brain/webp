// Package webp implements an WEBP image decoder based on libwebp compiled to WASM.
package webp

//go:generate make -C lib

import (
	"bytes"
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
	webpMaxHeaderSize     = 32
	webpDemuxABIVersion   = 0x0107
	webpDecoderABIVersion = 0x0209
	webpEncoderABIVersion = 0x020f
)

// WEBP represents the possibly multiple images stored in a WEBP file.
type WEBP struct {
	// Decoded images.
	Image []image.Image
	// Delay times, one per frame, in milliseconds.
	Delay []int
	// LoopCount is the number of times the animation repeats (0 = infinite).
	LoopCount int
}

// DefaultQuality is the default quality encoding parameter.
const DefaultQuality = 75

// DefaultMethod is the default method encoding parameter.
const DefaultMethod = 4

// Options are the encoding parameters, plus AutoRotate which applies to Decode.
type Options struct {
	// Quality in the range [0,100]. Default is 75.
	Quality int
	// Lossless enables lossless compression. Lossless ignores quality.
	Lossless bool
	// Method is quality/speed trade-off (0=fast, 6=slower-better). Default is 4.
	Method int
	// Exact preserve the exact RGB values in transparent area.
	Exact bool
	// AutoRotate applies the EXIF orientation to the decoded image (Decode/DecodeAll only).
	AutoRotate bool
}

// decodeWEBP dispatches to the dynamic (system libwebp) or wasm backend.
func decodeWEBP(r io.Reader, configOnly, decodeAll bool) (*WEBP, image.Config, error) {
	if dynamic {
		return decodeDynamic(r, configOnly, decodeAll)
	}

	return decode(r, configOnly, decodeAll)
}

// Decode reads a WEBP image from r; pass Options{AutoRotate: true} to apply the EXIF orientation.
func Decode(r io.Reader, opts ...Options) (image.Image, error) {
	if len(opts) > 0 && opts[0].AutoRotate {
		data, err := io.ReadAll(r)
		if err != nil {
			return nil, err
		}

		ret, _, err := decodeWEBP(bytes.NewReader(data), false, false)
		if err != nil {
			return nil, err
		}

		return applyOrientation(ret.Image[0], exifOrientation(data)), nil
	}

	ret, _, err := decodeWEBP(r, false, false)
	if err != nil {
		return nil, err
	}

	return ret.Image[0], nil
}

// DecodeConfig returns the color model and dimensions of a WEBP image without decoding the entire image.
func DecodeConfig(r io.Reader) (image.Config, error) {
	_, cfg, err := decodeWEBP(r, true, false)
	if err != nil {
		return image.Config{}, err
	}

	return cfg, nil
}

// DecodeAll returns the sequential frames and timing; pass Options{AutoRotate: true} to orient each frame.
func DecodeAll(r io.Reader, opts ...Options) (*WEBP, error) {
	if len(opts) > 0 && opts[0].AutoRotate {
		data, err := io.ReadAll(r)
		if err != nil {
			return nil, err
		}

		ret, _, err := decodeWEBP(bytes.NewReader(data), false, true)
		if err != nil {
			return nil, err
		}

		o := exifOrientation(data)
		for i := range ret.Image {
			ret.Image[i] = applyOrientation(ret.Image[i], o)
		}

		return ret, nil
	}

	ret, _, err := decodeWEBP(r, false, true)
	if err != nil {
		return nil, err
	}

	return ret, nil
}

// Encode writes the image m to w with the given options.
func Encode(w io.Writer, m image.Image, o ...Options) error {
	lossless := false
	quality := DefaultQuality
	method := DefaultMethod
	exact := false

	if o != nil {
		opt := o[0]
		lossless = opt.Lossless
		quality = opt.Quality
		method = opt.Method
		exact = opt.Exact

		if quality <= 0 {
			quality = DefaultQuality
		} else if quality > 100 {
			quality = 100
		}

		if method < 0 {
			method = DefaultMethod
		} else if method > 6 {
			method = 6
		}
	}

	if dynamic {
		err := encodeDynamic(w, m, quality, method, lossless, exact)
		if err != nil {
			return err
		}
	} else {
		err := encode(w, m, quality, method, lossless, exact)
		if err != nil {
			return err
		}
	}

	return nil
}

// EncodeAll writes the animation anim to w; all frames must share the same bounds.
func EncodeAll(w io.Writer, anim *WEBP, o ...Options) error {
	if anim == nil || len(anim.Image) == 0 {
		return ErrEncode
	}

	lossless := false
	quality := DefaultQuality
	method := DefaultMethod
	exact := false

	if o != nil {
		opt := o[0]
		lossless = opt.Lossless
		quality = opt.Quality
		method = opt.Method
		exact = opt.Exact

		if quality <= 0 {
			quality = DefaultQuality
		} else if quality > 100 {
			quality = 100
		}

		if method < 0 {
			method = DefaultMethod
		} else if method > 6 {
			method = 6
		}
	}

	b := anim.Image[0].Bounds()
	width, height := b.Dx(), b.Dy()
	frameSize := width * height * 4

	frames := make([]byte, frameSize*len(anim.Image))
	delays := make([]int, len(anim.Image))

	for i, img := range anim.Image {
		if img.Bounds().Dx() != width || img.Bounds().Dy() != height {
			return ErrEncode
		}

		dst := image.NewNRGBA(image.Rect(0, 0, width, height))
		draw.Draw(dst, dst.Bounds(), img, img.Bounds().Min, draw.Src)
		copy(frames[i*frameSize:(i+1)*frameSize], dst.Pix)

		if i < len(anim.Delay) {
			delays[i] = anim.Delay[i]
		}
	}

	data, err := encodeAnimation(frames, width, height, len(anim.Image), delays, anim.LoopCount, quality, method, lossless, exact)
	if err != nil {
		return err
	}

	_, err = w.Write(data)

	return err
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
	decodeWrapper := func(r io.Reader) (image.Image, error) {
		return Decode(r)
	}

	image.RegisterFormat("webp", "RIFF????WEBPVP8", decodeWrapper, DecodeConfig)
}
