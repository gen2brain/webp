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
BenchmarkDecodeWasm-8                            129	   9359457 ns/op	 1334194 B/op	      50 allocs/op
BenchmarkDecodeDynamic-8                         313	   3751982 ns/op	 1335973 B/op	      53 allocs/op
BenchmarkDecodeTranspiled-8 (1)                  138	   8562133 ns/op	 1335622 B/op	      52 allocs/op
BenchmarkDecodeCGo1-8 (2)                        300	   3897300 ns/op	 1333630 B/op	      21 allocs/op
BenchmarkDecodeCGo2-8 (3)                        314	   3801195 ns/op	 1334020 B/op	      22 allocs/op

BenchmarkDecodeConfigStd-8                   1653494	       712.9 ns/op	    3600 B/op	       6 allocs/op
BenchmarkDecodeConfigWasm-8                  1713109	       694.0 ns/op	     368 B/op	      20 allocs/op
BenchmarkDecodeConfigDynamic-8               2406602	       503.6 ns/op	     456 B/op	       9 allocs/op
BenchmarkDecodeConfigTranspiled-8 (1)        2568049	       450.0 ns/op	    3600 B/op	       6 allocs/op
BenchmarkDecodeConfigCGo1-8 (2)              8920566	       148.3 ns/op	     128 B/op	       3 allocs/op
BenchmarkDecodeConfigCGo2-8 (3)                24165	       50703 ns/op	  285376 B/op	      20 allocs/op

BenchmarkEncodeWasm-8                             10	 103065077 ns/op	  133804 B/op	      17 allocs/op
BenchmarkEncodeDynamic-8                          52	  22548953 ns/op	   26388 B/op	      13 allocs/op
BenchmarkEncodeTranspiled-8 (1)                   18	  60042805 ns/op	   76104 B/op	      36 allocs/op
BenchmarkEncodeCGo1-8 (2)                         31	  32538122 ns/op	 3213497 B/op	  524294 allocs/op
BenchmarkEncodeCGo2-8 (3)                         52	  22482704 ns/op	   26043 B/op	       5 allocs/op
```

- `(1)` [git.sr.ht/~jackmordaunt/go-libwebp](https://git.sr.ht/~jackmordaunt/go-libwebp)
- `(2)` [github.com/chai2010/webp](https://github.com/chai2010/webp)
- `(3)` [github.com/kolesa-team/go-webp](https://github.com/kolesa-team/go-webp)

