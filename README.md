## webp
[![Status](https://github.com/gen2brain/webp/actions/workflows/test.yml/badge.svg)](https://github.com/gen2brain/webp/actions)
[![Go Reference](https://pkg.go.dev/badge/github.com/gen2brain/webp.svg)](https://pkg.go.dev/github.com/gen2brain/webp)

Go decoder for [WebP Image File Format](https://en.wikipedia.org/wiki/WebP) with support for animated WebP images.

Based on [libwebp](https://github.com/webmproject/libwebp) compiled to [WASM](https://en.wikipedia.org/wiki/WebAssembly) and used with [wazero](https://wazero.io/) runtime (CGo-free).

The library will first try to use a dynamic/shared library (if installed) via [purego](https://github.com/ebitengine/purego) and will fall back to WASM.
