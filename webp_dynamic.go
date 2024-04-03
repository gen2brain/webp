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

	var err error
	var data []byte

	if configOnly {
		data = make([]byte, maxWebpHeaderSize)
		_, err = r.Read(data)
		if err != nil {
			return nil, cfg, err
		}
	} else {
		data, err = io.ReadAll(r)
		if err != nil {
			return nil, cfg, err
		}
	}

	width, height, ok := webpGetInfo(data)
	if !ok {
		return nil, cfg, ErrDecode
	}

	cfg.Width = width
	cfg.Height = height
	cfg.ColorModel = color.NYCbCrAModel

	if configOnly {
		return nil, cfg, nil
	}

	var wpData webpData
	wpData.Size = uint64(len(data))
	wpData.Bytes = &data[0]

	demuxer := webpDemux(&wpData)
	defer webpDemuxDelete(demuxer)

	delay := make([]int, 0)
	images := make([]*image.NYCbCrA, 0)

	var iter webpIterator
	defer webpDemuxReleaseIterator(&iter)

	var config webpDecoderConfig
	if !webpInitDecoderConfig(&config) {
		return nil, cfg, ErrDecode
	}

	config.Output.Colorspace = modeYUVA
	config.Options.UseThreads = 1

	if !webpDemuxGetFrame(demuxer, 1, &iter) {
		return nil, cfg, ErrDecode
	}

	rect := image.Rect(0, 0, cfg.Width, cfg.Height)

	for {
		ok = webpDecode(iter.Fragment.Bytes, iter.Fragment.Size, &config)
		if !ok {
			break
		}

		img := image.NewNYCbCrA(rect, image.YCbCrSubsampleRatio420)
		out := *(*webpYUVABuffer)(unsafe.Pointer(&config.Output.U))

		copy(img.Y, unsafe.Slice(out.Y, out.YSize))
		copy(img.Cb, unsafe.Slice(out.U, out.USize))
		copy(img.Cr, unsafe.Slice(out.V, out.VSize))
		copy(img.A, unsafe.Slice(out.A, out.ASize))

		images = append(images, img)
		delay = append(delay, int(iter.Duration))

		webpFreeDecBuffer(&config.Output)

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

	_, err := w.Write(unsafe.Slice(out, size))
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
	purego.RegisterLibFunc(&_webpDecode, libwebp, "WebPDecode")
	purego.RegisterLibFunc(&_webpGetInfo, libwebp, "WebPGetInfo")
	purego.RegisterLibFunc(&_webpInitDecoderConfig, libwebp, "WebPInitDecoderConfigInternal")
	purego.RegisterLibFunc(&_webpFree, libwebp, "WebPFree")
	purego.RegisterLibFunc(&_webpFreeDecBuffer, libwebp, "WebPFreeDecBuffer")
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
	_webpDecode               func(*uint8, uint64, *webpDecoderConfig) int
	_webpGetInfo              func(*uint8, uint64, *int, *int) int
	_webpInitDecoderConfig    func(*webpDecoderConfig) int
	_webpFree                 func(*uint8)
	_webpFreeDecBuffer        func(*webpDecBuffer)
	_webpEncodeRGBA           func(*uint8, int, int, int, float32, **uint8) uint64
	_webpEncodeLosslessRGBA   func(*uint8, int, int, int, **uint8) uint64
)

func webpDemux(data *webpData) *webpDemuxer {
	return _webpDemux(data, 0, nil, 0x0107) // WEBP_DEMUX_ABI_VERSION
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

func webpDecode(data *uint8, size uint64, config *webpDecoderConfig) bool {
	ret := _webpDecode(data, size, config)

	return ret == 0
}

func webpGetInfo(data []byte) (int, int, bool) {
	var width, height int

	ret := _webpGetInfo(&data[0], uint64(len(data)), &width, &height)
	b := ret != 0

	return width, height, b
}

func webpInitDecoderConfig(config *webpDecoderConfig) bool {
	ret := _webpInitDecoderConfig(config)

	return ret == 0
}

func webpFree(p *uint8) {
	_webpFree(p)
}

func webpFreeDecBuffer(p *webpDecBuffer) {
	_webpFreeDecBuffer(p)
}

func webpEncodeRGBA(data *uint8, width, height, stride int, quality float32, output **uint8) uint64 {
	return _webpEncodeRGBA(data, width, height, stride, quality, output)
}

func webpEncodeLosslessRGBA(data *uint8, width, height, stride int, output **uint8) uint64 {
	return _webpEncodeLosslessRGBA(data, width, height, stride, output)
}

const modeYUVA = 12

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

type webpDecoderOptions struct {
	BypassFiltering        int32
	NoFancyUpsampling      int32
	UseCropping            int32
	CropLeft               int32
	CropTop                int32
	CropWidth              int32
	CropHeight             int32
	UseScaling             int32
	ScaledWidth            int32
	ScaledHeight           int32
	UseThreads             int32
	DitheringStrength      int32
	Flip                   int32
	AlphaDitheringStrength int32
	_                      [5]uint32
}

type webpDecoderConfig struct {
	Input   webpBitstreamFeatures
	Output  webpDecBuffer
	Options webpDecoderOptions
	_       [4]byte
}

type webpDecBuffer struct {
	Colorspace       uint32
	Width            int32
	Height           int32
	IsExternalMemory int32
	U                [80]byte
	_                [4]uint32
	PrivateMemory    *uint8
}

type webpBitstreamFeatures struct {
	Width     int32
	Height    int32
	Alpha     int32
	Animation int32
	Format    int32
	_         [5]uint32
}

type webpYUVABuffer struct {
	Y       *uint8
	U       *uint8
	V       *uint8
	A       *uint8
	YStride int32
	UStride int32
	VStride int32
	AStride int32
	YSize   uint64
	USize   uint64
	VSize   uint64
	ASize   uint64
}
