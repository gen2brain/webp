//go:build windows && !nodynamic

package webp

import (
	"fmt"
	"syscall"
)

const (
	libname      = "libwebp.dll"
	libnameDemux = "libwebpdemux.dll"
)

func loadLibrary(name string) (uintptr, error) {
	handle, err := syscall.LoadLibrary(name)
	if err != nil {
		return 0, fmt.Errorf("cannot load library %s: %w", libname, err)
	}

	return uintptr(handle), nil
}
