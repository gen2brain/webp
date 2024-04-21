//go:build (!unix && !darwin && !windows) || nodynamic

package webp

import (
	"fmt"
	"image"
	"io"
)

var (
	dynamic    = false
	dynamicErr = fmt.Errorf("webp: dynamic disabled")
)

func decodeDynamic(r io.Reader, configOnly, decodeAll bool) (*WEBP, image.Config, error) {
	return nil, image.Config{}, dynamicErr
}

func encodeDynamic(w io.Writer, m image.Image, quality, method int, lossless, exact bool) error {
	return dynamicErr
}

func loadLibrary(name string) (uintptr, error) {
	return 0, dynamicErr
}
