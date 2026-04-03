//go:build unix && !darwin && !nodynamic

package webp

import (
	"debug/elf"
	"fmt"
	"os"
	"runtime"
	"strings"

	"github.com/ebitengine/purego"
)

const (
	libname      = "libwebp.so"
	libnameDemux = "libwebpdemux.so"
)

func loadLibrary(name string) (uintptr, error) {
	if runtime.GOOS == "linux" && !isDynamicBinary() && !isMusl() {
		return 0, fmt.Errorf("not a dynamic binary")
	}

	handle, err := purego.Dlopen(name, purego.RTLD_NOW|purego.RTLD_GLOBAL)
	if err != nil {
		return 0, fmt.Errorf("cannot load library: %w", err)
	}

	return handle, nil
}

func isDynamicBinary() bool {
	fileName, err := os.Executable()
	if err != nil {
		panic(err)
	}

	fl, err := elf.Open(fileName)
	if err != nil {
		panic(err)
	}

	defer fl.Close()

	_, err = fl.DynamicSymbols()
	if err == nil {
		return true
	}

	return false
}

// isMusl returns true if the current process is linked against musl libc.
// Musl supports dlopen from statically-linked binaries, unlike glibc.
func isMusl() bool {
	maps, err := os.ReadFile("/proc/self/maps")
	if err == nil && strings.Contains(string(maps), "musl") {
		return true
	}

	// For static binaries /proc/self/maps won't show musl.
	// Check if the musl dynamic linker exists on the system.
	for _, path := range []string{
		"/lib/ld-musl-aarch64.so.1",
		"/lib/ld-musl-x86_64.so.1",
		"/lib/ld-musl-armhf.so.1",
		"/lib/ld-musl-i386.so.1",
		"/lib/ld-musl-riscv64.so.1",
	} {
		if _, err := os.Stat(path); err == nil {
			return true
		}
	}

	return false
}
