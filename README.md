## webp
[![Status](https://github.com/gen2brain/webp/actions/workflows/test.yml/badge.svg)](https://github.com/gen2brain/webp/actions)
[![Go Reference](https://pkg.go.dev/badge/github.com/gen2brain/webp.svg)](https://pkg.go.dev/github.com/gen2brain/webp)

Go encoder/decoder for [WebP Image File Format](https://en.wikipedia.org/wiki/WebP) with support for animated WebP images (decode only).

Based on [libwebp](https://github.com/webmproject/libwebp) compiled to [WASM](https://en.wikipedia.org/wiki/WebAssembly) and transpiled to pure Go with [wasm2go](https://github.com/ncruces/wasm2go) (CGo-free).

The library will first try to use a dynamic/shared library (if installed) via [purego](https://github.com/ebitengine/purego) and will fall back to the transpiled Go.

### Build tags

* `nodynamic` - do not use dynamic/shared library (use only the transpiled Go)

