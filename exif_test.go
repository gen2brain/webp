package webp

import (
	"bytes"
	"os"
	"testing"
)

func TestDecodeExif(t *testing.T) {
	data, err := os.ReadFile("testdata/exif.webp")
	if err != nil {
		t.Fatal(err)
	}

	ex, err := DecodeExif(bytes.NewReader(data))
	if err != nil {
		t.Fatal(err)
	}

	if ex.Orientation != 6 {
		t.Errorf("Orientation = %d, want 6", ex.Orientation)
	}
	if ex.Make != "TestCam" {
		t.Errorf("Make = %q, want TestCam", ex.Make)
	}
	if ex.Model != "WebpEXIF" {
		t.Errorf("Model = %q, want WebpEXIF", ex.Model)
	}
	if ex.Software != "cbconvert" {
		t.Errorf("Software = %q, want cbconvert", ex.Software)
	}
}

func TestDecodeExifNone(t *testing.T) {
	data, err := os.ReadFile("testdata/test.webp")
	if err != nil {
		t.Fatal(err)
	}

	if _, err := DecodeExif(bytes.NewReader(data)); err != ErrNoExif {
		t.Errorf("err = %v, want ErrNoExif", err)
	}
}

func TestDecodeAutoRotate(t *testing.T) {
	data, err := os.ReadFile("testdata/exif.webp")
	if err != nil {
		t.Fatal(err)
	}

	plain, err := Decode(bytes.NewReader(data))
	if err != nil {
		t.Fatal(err)
	}
	if b := plain.Bounds(); b.Dx() != 512 || b.Dy() != 256 {
		t.Errorf("plain decode = %dx%d, want 512x256", b.Dx(), b.Dy())
	}

	// Orientation 6 (rotate 90 CW) swaps width and height.
	rot, err := Decode(bytes.NewReader(data), Options{AutoRotate: true})
	if err != nil {
		t.Fatal(err)
	}
	if b := rot.Bounds(); b.Dx() != 256 || b.Dy() != 512 {
		t.Errorf("auto-rotated decode = %dx%d, want 256x512", b.Dx(), b.Dy())
	}
}
