package webp

import (
	"bytes"
	_ "embed"
	"image"
	"image/jpeg"
	"io"
	"testing"
)

//go:embed testdata/test.webp
var testWebp []byte

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

func TestEncode(t *testing.T) {
	img, err := Decode(bytes.NewReader(testWebp))
	if err != nil {
		t.Fatal(err)
	}

	err = Encode(io.Discard, img)
	if err != nil {
		t.Fatal(err)
	}
}

func BenchmarkDecodeWebP(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_, _, err := decode(bytes.NewReader(testWebp), false, false)
		if err != nil {
			b.Error(err)
		}
	}
}

func BenchmarkDecodeWebPDynamic(b *testing.B) {
	if Dynamic() != nil {
		b.Errorf("dynamic/shared library not installed")
		return
	}

	for i := 0; i < b.N; i++ {
		_, _, err := decodeDynamic(bytes.NewReader(testWebp), false, false)
		if err != nil {
			b.Error(err)
		}
	}
}

func BenchmarkDecodeConfigWebP(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_, _, err := decode(bytes.NewReader(testWebp), true, false)
		if err != nil {
			b.Error(err)
		}
	}
}

func BenchmarkDecodeConfigWebPDynamic(b *testing.B) {
	if Dynamic() != nil {
		b.Errorf("dynamic/shared library not installed")
		return
	}

	for i := 0; i < b.N; i++ {
		_, _, err := decodeDynamic(bytes.NewReader(testWebp), true, false)
		if err != nil {
			b.Error(err)
		}
	}
}

func BenchmarkEncodeWebP(b *testing.B) {
	if Dynamic() != nil {
		b.Errorf("dynamic/shared library not installed")
		return
	}

	img, err := Decode(bytes.NewReader(testWebp))
	if err != nil {
		b.Fatal(err)
	}

	for i := 0; i < b.N; i++ {
		err := encode(io.Discard, img, 75, false)
		if err != nil {
			b.Error(err)
		}
	}
}

func BenchmarkEncodeWebPDynamic(b *testing.B) {
	if Dynamic() != nil {
		b.Errorf("dynamic/shared library not installed")
		return
	}

	img, err := Decode(bytes.NewReader(testWebp))
	if err != nil {
		b.Fatal(err)
	}

	for i := 0; i < b.N; i++ {
		err := encodeDynamic(io.Discard, img, 75, false)
		if err != nil {
			b.Error(err)
		}
	}
}
