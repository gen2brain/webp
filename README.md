## webp
[![Status](https://github.com/gen2brain/webp/actions/workflows/test.yml/badge.svg)](https://github.com/gen2brain/webp/actions)
[![Go Reference](https://pkg.go.dev/badge/github.com/gen2brain/webp.svg)](https://pkg.go.dev/github.com/gen2brain/webp)

Go encoder/decoder for [WebP Image File Format](https://en.wikipedia.org/wiki/WebP) with support for animated WebP images (decode only).

Based on [libwebp](https://github.com/webmproject/libwebp) compiled to [WASM](https://en.wikipedia.org/wiki/WebAssembly) and used with [wazero](https://wazero.io/) runtime (CGo-free).

The library will first try to use a dynamic/shared library (if installed) via [purego](https://github.com/ebitengine/purego) and will fall back to WASM.

### Benchmark

```
goos: linux
goarch: amd64
pkg: github.com/gen2brain/webp
cpu: 11th Gen Intel(R) Core(TM) i7-1185G7 @ 3.00GHz

BenchmarkDecodeStd-8                             157	   7639585 ns/op	  473683 B/op	      13 allocs/op
BenchmarkDecodeWasm-8                            169	   7038634 ns/op	  285712 B/op	      49 allocs/op
BenchmarkDecodeDynamic-8                         344	   3497863 ns/op	  943356 B/op	      58 allocs/op
BenchmarkDecodeTranspiled-8 (1)                  138	   8562133 ns/op	 1335622 B/op	      52 allocs/op
BenchmarkDecodeCGo1-8 (2)                        300	   3897300 ns/op	 1333630 B/op	      21 allocs/op
BenchmarkDecodeCGo2-8 (3)                        314	   3801195 ns/op	 1334020 B/op	      22 allocs/op

BenchmarkEncodeWasm-8                             13	  88419790 ns/op	   72773 B/op	      16 allocs/op
BenchmarkEncodeDynamic-8                          55	  19022243 ns/op	   19888 B/op	      42 allocs/op
BenchmarkEncodeTranspiled-8 (1)                   18	  60042805 ns/op	   76104 B/op	      36 allocs/op
BenchmarkEncodeCGo1-8 (2)                         31	  32538122 ns/op	 3213497 B/op	  524294 allocs/op
BenchmarkEncodeCGo2-8 (3)                         52	  22482704 ns/op	   26043 B/op	       5 allocs/op
```

- `(1)` [git.sr.ht/~jackmordaunt/go-libwebp](https://git.sr.ht/~jackmordaunt/go-libwebp)
- `(2)` [github.com/chai2010/webp](https://github.com/chai2010/webp)
- `(3)` [github.com/kolesa-team/go-webp](https://github.com/kolesa-team/go-webp)

