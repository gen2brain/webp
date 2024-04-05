package webp

import (
	"bytes"
	_ "embed"
	"fmt"
	"image"
	"image/jpeg"
	"image/png"
	"io"
	"testing"
)

//go:embed testdata/test.webp
var testWebp []byte

//go:embed testdata/test.png
var testPng []byte

//go:embed testdata/anim.webp
var testWebpAnim []byte

func TestDecode(t *testing.T) {
	img, err := Decode(bytes.NewReader(testWebp))
	if err != nil {
		t.Fatal(err)
	}

	err = jpeg.Encode(io.Discard, img, nil)
	if err != nil {
		t.Error(err)
	}
}

func TestDecodeWasm(t *testing.T) {
	img, _, err := decode(bytes.NewReader(testWebp), false, false)
	if err != nil {
		t.Fatal(err)
	}

	err = jpeg.Encode(io.Discard, img.Image[0], nil)
	if err != nil {
		t.Error(err)
	}
}

func TestDecodeDynamic(t *testing.T) {
	if err := Dynamic(); err != nil {
		fmt.Println(err)
		t.Skip()
	}

	img, _, err := decodeDynamic(bytes.NewReader(testWebp), false, false)
	if err != nil {
		t.Fatal(err)
	}

	err = jpeg.Encode(io.Discard, img.Image[0], nil)
	if err != nil {
		t.Error(err)
	}
}

func TestDecodeAnim(t *testing.T) {
	ret, err := DecodeAll(bytes.NewReader(testWebpAnim))
	if err != nil {
		t.Fatal(err)
	}

	if len(ret.Image) != len(ret.Delay) {
		t.Errorf("got %d, want %d", len(ret.Delay), len(ret.Image))
	}

	if len(ret.Image) != 17 {
		t.Errorf("got %d, want %d", len(ret.Image), 17)
	}

	for _, img := range ret.Image {
		err = jpeg.Encode(io.Discard, img, nil)
		if err != nil {
			t.Error(err)
		}
	}
}

func TestImageDecode(t *testing.T) {
	img, _, err := image.Decode(bytes.NewReader(testWebp))
	if err != nil {
		t.Fatal(err)
	}

	err = jpeg.Encode(io.Discard, img, nil)
	if err != nil {
		t.Error(err)
	}
}

func TestDecodeConfig(t *testing.T) {
	cfg, err := DecodeConfig(bytes.NewReader(testWebp))
	if err != nil {
		t.Fatal(err)
	}

	if cfg.Width != 512 {
		t.Errorf("width: got %d, want %d", cfg.Width, 512)
	}

	if cfg.Height != 512 {
		t.Errorf("height: got %d, want %d", cfg.Height, 512)
	}
}

func TestEncodeRGBA(t *testing.T) {
	img, err := png.Decode(bytes.NewReader(testPng))
	if err != nil {
		t.Fatal(err)
	}

	err = Encode(io.Discard, img)
	if err != nil {
		t.Fatal(err)
	}
}

func TestEncodeWasm(t *testing.T) {
	img, err := Decode(bytes.NewReader(testWebp))
	if err != nil {
		t.Fatal(err)
	}

	err = encode(io.Discard, img, DefaultQuality, DefaultMethod, false, false)
	if err != nil {
		t.Fatal(err)
	}
}

func TestEncodeDynamic(t *testing.T) {
	if err := Dynamic(); err != nil {
		fmt.Println(err)
		t.Skip()
	}

	img, err := Decode(bytes.NewReader(testWebp))
	if err != nil {
		t.Fatal(err)
	}

	err = encodeDynamic(io.Discard, img, DefaultQuality, DefaultMethod, false, false)
	if err != nil {
		t.Fatal(err)
	}
}

func BenchmarkDecodeWasm(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_, _, err := decode(bytes.NewReader(testWebp), false, false)
		if err != nil {
			b.Error(err)
		}
	}
}

func BenchmarkDecodeDynamic(b *testing.B) {
	if err := Dynamic(); err != nil {
		fmt.Println(err)
		b.Skip()
	}

	for i := 0; i < b.N; i++ {
		_, _, err := decodeDynamic(bytes.NewReader(testWebp), false, false)
		if err != nil {
			b.Error(err)
		}
	}
}

func BenchmarkEncodeWasm(b *testing.B) {
	img, err := Decode(bytes.NewReader(testWebp))
	if err != nil {
		b.Fatal(err)
	}

	for i := 0; i < b.N; i++ {
		err = encode(io.Discard, img, DefaultQuality, DefaultMethod, false, false)
		if err != nil {
			b.Error(err)
		}
	}
}

func BenchmarkEncodeDynamic(b *testing.B) {
	if err := Dynamic(); err != nil {
		fmt.Println(err)
		b.Skip()
	}

	img, err := Decode(bytes.NewReader(testWebp))
	if err != nil {
		b.Fatal(err)
	}

	for i := 0; i < b.N; i++ {
		err = encodeDynamic(io.Discard, img, DefaultQuality, DefaultMethod, false, false)
		if err != nil {
			b.Error(err)
		}
	}
}
