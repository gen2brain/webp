package webp

import (
	"fmt"
	"image"
	"image/color"
	"io"
	"runtime"
	"unsafe"

	"github.com/ebitengine/purego"
)

func decodeDynamic(r io.Reader, configOnly, decodeAll bool) (*WEBP, image.Config, error) {
	var cfg image.Config

	data, err := io.ReadAll(r)
	if err != nil {
		return nil, cfg, err
	}

	width, height, ok := webpGetInfo(data)
	if !ok {
		return nil, cfg, ErrDecode
	}

	cfg.Width = width
	cfg.Height = height
	cfg.ColorModel = color.NRGBAModel

	if configOnly {
		return nil, cfg, nil
	}

	var wpData webpData
	wpData.Size = uint64(len(data))
	wpData.Bytes = &data[0]

	demuxer := webpDemux(&wpData)
	defer webpDemuxDelete(demuxer)

	delay := make([]int, 0)
	images := make([]*image.NRGBA, 0)

	var iter webpIterator
	defer webpDemuxReleaseIterator(&iter)

	ok = webpDemuxGetFrame(demuxer, 1, &iter)
	if !ok {
		return nil, cfg, ErrDecode
	}

	size := cfg.Width * cfg.Height * 4

	for {
		img := image.NewNRGBA(image.Rect(0, 0, cfg.Width, cfg.Height))
		decoded := webpDecodeRGBA(iter.Fragment.Bytes, iter.Fragment.Size)

		copy(img.Pix, unsafe.Slice(decoded, size))
		images = append(images, img)

		delay = append(delay, int(iter.Duration))

		webpFree(decoded)

		if !decodeAll {
			break
		}

		ok = webpDemuxNextFrame(&iter)
		if !ok {
			break
		}
	}

	runtime.KeepAlive(data)

	ret := &WEBP{
		Image: images,
		Delay: delay,
	}

	return ret, cfg, nil
}

func encodeDynamic(w io.Writer, m image.Image, quality int, lossless bool) error {
	img := imageToNRGBA(m)

	out := new(uint8)
	pix := unsafe.SliceData(img.Pix)

	var size uint64
	if lossless {
		size = webpEncodeLosslessRGBA(pix, img.Bounds().Dx(), img.Bounds().Dy(), img.Stride, &out)
		if size == 0 {
			return ErrEncode
		}
	} else {
		size = webpEncodeRGBA(pix, img.Bounds().Dx(), img.Bounds().Dy(), img.Stride, float32(quality), &out)
		if size == 0 {
			return ErrEncode
		}
	}

	if out == nil {
		return ErrEncode
	}

	defer webpFree(out)

	buf := unsafe.Slice(out, size)

	_, err := w.Write(buf)
	if err != nil {
		return fmt.Errorf("write: %w", err)
	}

	return nil
}

func init() {
	var err error
	defer func() {
		if r := recover(); r != nil {
			dynamic = false
			dynamicErr = fmt.Errorf("%v", r)
		}
	}()

	libwebp, err = loadLibrary(libname)
	if err == nil {
		libwebpDemux, err = loadLibrary(libnameDemux)
		if err == nil {
			dynamic = true
		} else {
			dynamicErr = err
		}
	} else {
		dynamicErr = err
	}

	if !dynamic {
		return
	}

	purego.RegisterLibFunc(&_webpDemux, libwebpDemux, "WebPDemuxInternal")
	purego.RegisterLibFunc(&_webpDemuxDelete, libwebpDemux, "WebPDemuxDelete")
	purego.RegisterLibFunc(&_webpDemuxReleaseIterator, libwebpDemux, "WebPDemuxReleaseIterator")
	purego.RegisterLibFunc(&_webpDemuxNextFrame, libwebpDemux, "WebPDemuxNextFrame")
	purego.RegisterLibFunc(&_webpDemuxGetFrame, libwebpDemux, "WebPDemuxGetFrame")
	purego.RegisterLibFunc(&_webpDecodeRGBA, libwebp, "WebPDecodeRGBA")
	purego.RegisterLibFunc(&_webpGetInfo, libwebp, "WebPGetInfo")
	purego.RegisterLibFunc(&_webpFree, libwebp, "WebPFree")
	purego.RegisterLibFunc(&_webpEncodeRGBA, libwebp, "WebPEncodeRGBA")
	purego.RegisterLibFunc(&_webpEncodeLosslessRGBA, libwebp, "WebPEncodeLosslessRGBA")
}

var (
	libwebp      uintptr
	libwebpDemux uintptr
	dynamic      bool
	dynamicErr   error
)

var (
	_webpDemux                func(*webpData, int, *int, int) *webpDemuxer
	_webpDemuxDelete          func(*webpDemuxer)
	_webpDemuxReleaseIterator func(*webpIterator)
	_webpDemuxNextFrame       func(*webpIterator) int
	_webpDemuxGetFrame        func(*webpDemuxer, int, *webpIterator) int
	_webpDecodeRGBA           func(*uint8, uint64, *int, *int) *uint8
	_webpGetInfo              func(*uint8, uint64, *int, *int) int
	_webpFree                 func(*uint8)
	_webpEncodeRGBA           func(*uint8, int, int, int, float32, **uint8) uint64
	_webpEncodeLosslessRGBA   func(*uint8, int, int, int, **uint8) uint64
)

func webpDemux(data *webpData) *webpDemuxer {
	return _webpDemux(data, 0, nil, 0x0107)
}

func webpDemuxDelete(demuxer *webpDemuxer) {
	_webpDemuxDelete(demuxer)
}

func webpDemuxReleaseIterator(iterator *webpIterator) {
	_webpDemuxReleaseIterator(iterator)
}

func webpDemuxNextFrame(iterator *webpIterator) bool {
	ret := _webpDemuxNextFrame(iterator)

	return ret != 0
}

func webpDemuxGetFrame(demuxer *webpDemuxer, frameNumber int, iterator *webpIterator) bool {
	ret := _webpDemuxGetFrame(demuxer, frameNumber, iterator)

	return ret != 0
}

func webpDecodeRGBA(data *uint8, size uint64) *uint8 {
	return _webpDecodeRGBA(data, size, nil, nil)
}

func webpGetInfo(data []byte) (int, int, bool) {
	var width, height int

	ret := _webpGetInfo(&data[0], uint64(len(data)), &width, &height)
	b := ret != 0

	return width, height, b
}

func webpFree(p *uint8) {
	_webpFree(p)
}

func webpEncodeRGBA(data *uint8, width, height, stride int, quality float32, output **uint8) uint64 {
	return _webpEncodeRGBA(data, width, height, stride, quality, output)
}

func webpEncodeLosslessRGBA(data *uint8, width, height, stride int, output **uint8) uint64 {
	return _webpEncodeLosslessRGBA(data, width, height, stride, output)
}

type webpDemuxer struct{}

type webpData struct {
	Bytes *uint8
	Size  uint64
}

type webpIterator struct {
	FrameNum      int32
	NumFrames     int32
	XOffset       int32
	YOffset       int32
	Width         int32
	Height        int32
	Duration      int32
	DisposeMethod uint32
	Complete      int32
	Fragment      webpData
	HasAlpha      int32
	BlendMethod   uint32
	_             [2]uint32
	_             *byte
}
