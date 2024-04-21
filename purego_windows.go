//go:build windows && !nodynamic

package webp

import (
	"fmt"

	"golang.org/x/sys/windows"
)

const (
	libname      = "libwebp.dll"
	libnameDemux = "libwebpdemux.dll"
)

func loadLibrary(name string) (uintptr, error) {
	handle, err := windows.LoadLibrary(name)
	if err != nil {
		return 0, fmt.Errorf("cannot load library %s: %w", libname, err)
	}

	return uintptr(handle), nil
}
