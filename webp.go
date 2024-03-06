// Package webp implements an WEBP image decoder based on libwebp compiled to WASM.
package webp

import (
	"image"
)

// WEBP represents the possibly multiple images stored in a WEBP file.
type WEBP struct {
	// Decoded images.
	Image []image.Image
	// Delay times, one per frame, in milliseconds.
	Delay []int
}

func init() {
	image.RegisterFormat("webp", "RIFF????WEBPVP8", Decode, DecodeConfig)
}
