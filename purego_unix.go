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
	if runtime.GOOS == "linux" && !canDlopen() {
		return 0, fmt.Errorf("not a dynamic binary")
	}

	handle, err := purego.Dlopen(name, purego.RTLD_NOW|purego.RTLD_GLOBAL)
	if err != nil {
		return 0, fmt.Errorf("cannot load library: %w", err)
	}

	return handle, nil
}

// canDlopen reports whether dlopen is safe to call from this binary.
// Dynamic binaries can always dlopen. Static binaries can only dlopen
// safely if they were linked against musl libc, which has a fully
// functional dlopen implementation even in static executables.
// On glibc, dlopen from static binaries is not supported and may crash.
func canDlopen() bool {
	fileName, err := os.Executable()
	if err != nil {
		return false
	}

	fl, err := elf.Open(fileName)
	if err != nil {
		return false
	}
	defer fl.Close()

	// Dynamic binaries can always dlopen
	if _, err := fl.DynamicSymbols(); err == nil {
		return true
	}

	// Static binary — check if it was linked with musl by reading the
	// ELF .comment section, which contains the compiler/linker identification.
	// Musl-linked binaries typically have no .comment or have "GCC" without
	// glibc markers. More reliably, we look for "musl" in the ELF interpreter
	// (PT_INTERP program header), which musl's linker sets even for static-pie.
	for _, prog := range fl.Progs {
		if prog.Type == elf.PT_INTERP {
			interp := make([]byte, prog.Filesz)
			if _, err := prog.ReadAt(interp, 0); err == nil {
				if strings.Contains(string(interp), "musl") {
					return true
				}
			}
			break
		}
	}

	return false
}
