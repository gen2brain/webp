//go:build unix && !darwin && !nodynamic

package webp

import (
	"fmt"

	"github.com/ebitengine/purego"
)

const (
	libname      = "libwebp.so"
	libnameDemux = "libwebpdemux.so"
)

func loadLibrary(name string) (uintptr, error) {
	handle, err := purego.Dlopen(name, purego.RTLD_NOW|purego.RTLD_GLOBAL)
	if err != nil {
		return 0, fmt.Errorf("cannot load library: %w", err)
	}

	return uintptr(handle), nil
}
