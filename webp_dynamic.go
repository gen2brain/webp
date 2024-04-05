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
		data = make([]byte, webpMaxHeaderSize)
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

func encodeDynamic(w io.Writer, m image.Image, quality, method int, lossless, exact bool) error {
	var config webpConfig
	if !webpConfigInit(&config) {
		return ErrEncode
	}

	config.Quality = float32(quality)
	config.ThreadLevel = 1
	config.Method = int32(method)

	config.Lossless = 0
	if lossless {
		config.Lossless = 1
	}

	config.Exact = 0
	if exact {
		config.Exact = 1
	}

	var picture webpPicture
	if !webpPictureInit(&picture) {
		return ErrEncode
	}
	defer webpPictureFree(&picture)

	picture.Width = int32(m.Bounds().Dx())
	picture.Height = int32(m.Bounds().Dy())

	var data []byte

	switch img := m.(type) {
	case *image.YCbCr:
		i := imageToNRGBA(img)
		data = i.Pix
		picture.UseArgb = 1
		picture.ArgbStride = int32(i.Stride)
	case *image.NYCbCrA:
		if img.SubsampleRatio == image.YCbCrSubsampleRatio420 {
			picture.Y = unsafe.SliceData(img.Y)
			picture.U = unsafe.SliceData(img.Cb)
			picture.V = unsafe.SliceData(img.Cr)
			picture.A = unsafe.SliceData(img.A)
			picture.YStride = int32(img.YStride)
			picture.UvStride = int32(img.CStride)
			picture.AStride = int32(img.AStride)
			picture.UseArgb = 0
			picture.Colorspace = 4 // WEBP_YUV420A
		} else {
			i := imageToNRGBA(img)
			data = i.Pix
			picture.UseArgb = 1
			picture.ArgbStride = int32(i.Stride)
		}
	case *image.RGBA:
		data = img.Pix
		picture.UseArgb = 1
		picture.ArgbStride = int32(img.Stride)
	case *image.NRGBA:
		data = img.Pix
		picture.UseArgb = 1
		picture.ArgbStride = int32(img.Stride)
	default:
		i := imageToNRGBA(img)
		data = i.Pix
		picture.UseArgb = 1
		picture.ArgbStride = int32(i.Stride)
	}

	if picture.UseArgb == 1 {
		if !webpPictureImportRGBA(&picture, unsafe.SliceData(data), int(picture.ArgbStride)) {
			return ErrEncode
		}
	}

	writeCallback := func(d *uint8, size uint64, picture *webpPicture) int {
		_, err := w.Write(unsafe.Slice(d, size))
		if err != nil {
			return 0
		}

		return 1
	}

	var writer webpMemoryWriter
	webpMemoryWriterInit(&writer)
	defer webpMemoryWriterClear(&writer)

	picture.Writer = purego.NewCallback(writeCallback)
	picture.CustomPtr = (*byte)(unsafe.Pointer(&writer))

	if !webpEncode(&config, &picture) {
		return ErrEncode
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
	purego.RegisterLibFunc(&_webpInitDecoderConfig, libwebp, "WebPInitDecoderConfigInternal")
	purego.RegisterLibFunc(&_webpGetInfo, libwebp, "WebPGetInfo")
	purego.RegisterLibFunc(&_webpPictureImportRGBA, libwebp, "WebPPictureImportRGBA")
	purego.RegisterLibFunc(&_webpConfigInit, libwebp, "WebPConfigInitInternal")
	purego.RegisterLibFunc(&_webpPictureInit, libwebp, "WebPPictureInitInternal")
	purego.RegisterLibFunc(&_webpPictureFree, libwebp, "WebPPictureFree")
	purego.RegisterLibFunc(&_webpMemoryWriterInit, libwebp, "WebPMemoryWriterInit")
	purego.RegisterLibFunc(&_webpMemoryWriterClear, libwebp, "WebPMemoryWriterClear")
	purego.RegisterLibFunc(&_webpFreeDecBuffer, libwebp, "WebPFreeDecBuffer")
	purego.RegisterLibFunc(&_webpEncode, libwebp, "WebPEncode")
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
	_webpInitDecoderConfig    func(*webpDecoderConfig) int
	_webpGetInfo              func(*uint8, uint64, *int, *int) int
	_webpPictureImportRGBA    func(*webpPicture, *uint8, int) int
	_webpConfigInit           func(*webpConfig, int, float32, int) int
	_webpPictureInit          func(*webpPicture, int) int
	_webpPictureFree          func(*webpPicture)
	_webpMemoryWriterInit     func(*webpMemoryWriter)
	_webpMemoryWriterClear    func(*webpMemoryWriter)
	_webpFreeDecBuffer        func(*webpDecBuffer)
	_webpEncode               func(*webpConfig, *webpPicture) int
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

func webpInitDecoderConfig(config *webpDecoderConfig) bool {
	ret := _webpInitDecoderConfig(config)

	return ret == 0
}

func webpGetInfo(data []byte) (int, int, bool) {
	var width, height int

	ret := _webpGetInfo(&data[0], uint64(len(data)), &width, &height)
	b := ret != 0

	return width, height, b
}

func webpPictureImportRGBA(picture *webpPicture, in *uint8, stride int) bool {
	ret := _webpPictureImportRGBA(picture, in, stride)

	return ret != 0
}

func webpConfigInit(config *webpConfig) bool {
	ret := _webpConfigInit(config, 0, DefaultQuality, webpEncoderABIVersion)

	return ret != 0
}

func webpPictureInit(picture *webpPicture) bool {
	ret := _webpPictureInit(picture, webpEncoderABIVersion)

	return ret != 0
}

func webpPictureFree(picture *webpPicture) {
	_webpPictureFree(picture)
}

func webpMemoryWriterInit(writer *webpMemoryWriter) {
	_webpMemoryWriterInit(writer)
}

func webpMemoryWriterClear(writer *webpMemoryWriter) {
	_webpMemoryWriterClear(writer)
}

func webpFreeDecBuffer(p *webpDecBuffer) {
	_webpFreeDecBuffer(p)
}

func webpEncode(config *webpConfig, picture *webpPicture) bool {
	ret := _webpEncode(config, picture)

	return ret != 0
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

type webpPicture struct {
	UseArgb       int32
	Colorspace    uint32
	Width         int32
	Height        int32
	Y             *uint8
	U             *uint8
	V             *uint8
	YStride       int32
	UvStride      int32
	A             *uint8
	AStride       int32
	Pad1          [2]uint32
	Argb          *uint32
	ArgbStride    int32
	Pad2          [3]uint32
	Writer        uintptr
	CustomPtr     *byte
	ExtraInfoType int32
	ExtraInfo     *uint8
	Stats         *webpAuxStats
	ErrorCode     uint32
	ProgressHook  *[0]byte
	UserData      *byte
	Pad3          [3]uint32
	Pad4          *uint8
	Pad5          *uint8
	Pad6          [8]uint32
	Memory_       *byte
	MemoryArgb    *byte
	Pad7          [2]*byte
}

type webpConfig struct {
	Lossless         int32
	Quality          float32
	Method           int32
	ImageHint        uint32
	TargetSize       int32
	TargetPsnr       float32
	Segments         int32
	SnsStrength      int32
	FilterStrength   int32
	FilterSharpness  int32
	FilterType       int32
	Autofilter       int32
	AlphaCompression int32
	AlphaFiltering   int32
	AlphaQuality     int32
	Pass             int32
	ShowCompressed   int32
	Preprocessing    int32
	Partitions       int32
	PartitionLimit   int32
	EmulateJpegSize  int32
	ThreadLevel      int32
	LowMemory        int32
	NearLossless     int32
	Exact            int32
	UseDeltaPalette  int32
	UseSharpYuv      int32
	Qmin             int32
	Qmax             int32
}

type webpAuxStats struct {
	CodedSize        int32
	PSNR             [5]float32
	BlockCount       [3]int32
	HeaderBytes      [2]int32
	ResidualBytes    [3][4]int32
	SegmentSize      [4]int32
	SegmentQuant     [4]int32
	SegmentLevel     [4]int32
	AlphaDataSize    int32
	LayerDataSize    int32
	LosslessFeatures uint32
	HistogramBits    int32
	TransformBits    int32
	CacheBits        int32
	PaletteSize      int32
	LosslessSize     int32
	LosslessHdrSize  int32
	LosslessDataSize int32
	Pad              [2]uint32
}

type webpMemoryWriter struct {
	Mem     *uint8
	Size    uint64
	MaxSize uint64
	Pad     [1]uint32
	_       [4]byte
}
