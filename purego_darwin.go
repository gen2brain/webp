//go:build darwin && !nodynamic

package webp

import (
	"fmt"

	"github.com/ebitengine/purego"
)

const (
	libname      = "libwebp.dylib"
	libnameDemux = "libwebpdemux.dylib"
)

func loadLibrary(name string) (uintptr, error) {
	handle, err := purego.Dlopen(name, purego.RTLD_NOW|purego.RTLD_GLOBAL)
	if err != nil {
		return 0, fmt.Errorf("cannot load library: %w", err)
	}

	return uintptr(handle), nil
}
