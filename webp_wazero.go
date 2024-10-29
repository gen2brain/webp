package webp

import (
	"bytes"
	"compress/gzip"
	"context"
	_ "embed"
	"fmt"
	"image"
	"image/color"
	"io"
	"os"
	"sync"
	"unsafe"

	"github.com/tetratelabs/wazero"
	"github.com/tetratelabs/wazero/imports/wasi_snapshot_preview1"
)

//go:embed lib/webp.wasm.gz
var webpWasm []byte

func decode(r io.Reader, configOnly, decodeAll bool) (*WEBP, image.Config, error) {
	initOnce()

	var cfg image.Config
	var data []byte

	ctx := context.Background()

	mod, err := rt.InstantiateModule(ctx, cm, mc)
	if err != nil {
		return nil, cfg, err
	}

	defer mod.Close(ctx)

	_alloc := mod.ExportedFunction("malloc")
	_free := mod.ExportedFunction("free")
	_decode := mod.ExportedFunction("decode")

	if configOnly {
		data = make([]byte, webpMaxHeaderSize)
		_, err = r.Read(data)
		if err != nil {
			return nil, cfg, fmt.Errorf("read: %w", err)
		}
	} else {
		data, err = io.ReadAll(r)
		if err != nil {
			return nil, cfg, fmt.Errorf("read: %w", err)
		}
	}

	inSize := len(data)

	res, err := _alloc.Call(ctx, uint64(inSize))
	if err != nil {
		return nil, cfg, fmt.Errorf("alloc: %w", err)
	}
	inPtr := res[0]
	defer _free.Call(ctx, inPtr)

	ok := mod.Memory().Write(uint32(inPtr), data)
	if !ok {
		return nil, cfg, ErrMemWrite
	}

	res, err = _alloc.Call(ctx, 4*4)
	if err != nil {
		return nil, cfg, fmt.Errorf("alloc: %w", err)
	}
	defer _free.Call(ctx, res[0])

	widthPtr := res[0]
	heightPtr := res[0] + 4
	countPtr := res[0] + 8
	animPtr := res[0] + 12

	all := 0
	if decodeAll {
		all = 1
	}

	res, err = _decode.Call(ctx, inPtr, uint64(inSize), 1, uint64(all), widthPtr, heightPtr, countPtr, animPtr, 0, 0)
	if err != nil {
		return nil, cfg, fmt.Errorf("decode: %w", err)
	}

	if res[0] == 0 {
		return nil, cfg, ErrDecode
	}

	width, ok := mod.Memory().ReadUint32Le(uint32(widthPtr))
	if !ok {
		return nil, cfg, ErrMemRead
	}

	height, ok := mod.Memory().ReadUint32Le(uint32(heightPtr))
	if !ok {
		return nil, cfg, ErrMemRead
	}

	anim, ok := mod.Memory().ReadUint32Le(uint32(animPtr))
	if !ok {
		return nil, cfg, ErrMemRead
	}

	hasAnimation := anim != 0

	cfg.Width = int(width)
	cfg.Height = int(height)

	cfg.ColorModel = color.NYCbCrAModel
	if hasAnimation {
		cfg.ColorModel = color.RGBAModel
	}

	if configOnly {
		return nil, cfg, nil
	}

	delay := make([]int, 0)
	images := make([]image.Image, 0)

	if decodeAll || hasAnimation {
		count, ok := mod.Memory().ReadUint32Le(uint32(countPtr))
		if !ok {
			return nil, cfg, ErrMemRead
		}

		size := cfg.Width * cfg.Height * 4

		res, err = _alloc.Call(ctx, uint64(size*int(count)))
		if err != nil {
			return nil, cfg, fmt.Errorf("alloc: %w", err)
		}
		outPtr := res[0]
		defer _free.Call(ctx, outPtr)

		res, err = _alloc.Call(ctx, uint64(4*int(count)))
		if err != nil {
			return nil, cfg, fmt.Errorf("alloc: %w", err)
		}
		delayPtr := res[0]
		defer _free.Call(ctx, delayPtr)

		res, err = _decode.Call(ctx, inPtr, uint64(inSize), 0, uint64(all), widthPtr, heightPtr, countPtr, animPtr, delayPtr, outPtr)
		if err != nil {
			return nil, cfg, fmt.Errorf("decode: %w", err)
		}

		if res[0] == 0 {
			return nil, cfg, ErrDecode
		}

		for i := 0; i < int(count); i++ {
			out, ok := mod.Memory().Read(uint32(outPtr)+uint32(i*size), uint32(size))
			if !ok {
				return nil, cfg, ErrMemRead
			}

			img := image.NewRGBA(image.Rect(0, 0, cfg.Width, cfg.Height))
			img.Pix = out

			images = append(images, img)

			d, ok := mod.Memory().ReadUint32Le(uint32(delayPtr) + uint32(i*4))
			if !ok {
				return nil, cfg, ErrMemRead
			}

			delay = append(delay, int(d))
		}

		ret := &WEBP{
			Image: images,
			Delay: delay,
		}

		return ret, cfg, nil
	}

	rect := image.Rect(0, 0, cfg.Width, cfg.Height)
	w, h := rect.Dx(), rect.Dy()
	cw := (rect.Max.X+1)/2 - rect.Min.X/2
	ch := (rect.Max.Y+1)/2 - rect.Min.Y/2

	i0 := 1*w*h + 0*cw*ch
	i1 := 1*w*h + 1*cw*ch
	i2 := 1*w*h + 2*cw*ch
	i3 := 2*w*h + 2*cw*ch

	size := i3

	res, err = _alloc.Call(ctx, uint64(size))
	if err != nil {
		return nil, cfg, fmt.Errorf("alloc: %w", err)
	}
	outPtr := res[0]
	defer _free.Call(ctx, outPtr)

	res, err = _decode.Call(ctx, inPtr, uint64(inSize), 0, uint64(all), widthPtr, heightPtr, countPtr, animPtr, 0, outPtr)
	if err != nil {
		return nil, cfg, fmt.Errorf("decode: %w", err)
	}

	if res[0] == 0 {
		return nil, cfg, ErrDecode
	}

	out, ok := mod.Memory().Read(uint32(outPtr), uint32(size))
	if !ok {
		return nil, cfg, ErrMemRead
	}

	img := &image.NYCbCrA{
		YCbCr: image.YCbCr{
			Y:              out[:i0:i0],
			Cb:             out[i0:i1:i1],
			Cr:             out[i1:i2:i2],
			SubsampleRatio: image.YCbCrSubsampleRatio420,
			YStride:        w,
			CStride:        cw,
			Rect:           rect,
		},
		A:       out[i2:],
		AStride: w,
	}

	images = append(images, img)

	ret := &WEBP{
		Image: images,
		Delay: delay,
	}

	return ret, cfg, nil
}

func encode(w io.Writer, m image.Image, quality, method int, lossless, exact bool) error {
	initOnce()

	ctx := context.Background()

	mod, err := rt.InstantiateModule(ctx, cm, mc)
	if err != nil {
		return err
	}

	defer mod.Close(ctx)

	_alloc := mod.ExportedFunction("malloc")
	_free := mod.ExportedFunction("free")
	_encode := mod.ExportedFunction("encode")

	var data []byte
	var colorspace int

	var width = m.Bounds().Dx()
	var height = m.Bounds().Dy()

	switch img := m.(type) {
	case *image.YCbCr:
		i := imageToNRGBA(img)
		data = i.Pix
	case *image.NYCbCrA:
		if img.SubsampleRatio == image.YCbCrSubsampleRatio420 {
			length := len(img.Y) + len(img.Cb) + len(img.Cr) + len(img.A)
			var b = struct {
				addr *uint8
				len  int
				cap  int
			}{&img.Y[0], length, length}
			data = *(*[]byte)(unsafe.Pointer(&b))
			colorspace = 4 // WEBP_YUV420A
		} else {
			i := imageToNRGBA(img)
			data = i.Pix
		}
	case *image.RGBA:
		data = img.Pix
	case *image.NRGBA:
		data = img.Pix
	default:
		i := imageToNRGBA(img)
		data = i.Pix
	}

	res, err := _alloc.Call(ctx, uint64(len(data)))
	if err != nil {
		return fmt.Errorf("alloc: %w", err)
	}
	inPtr := res[0]
	defer _free.Call(ctx, inPtr)

	ok := mod.Memory().Write(uint32(inPtr), data)
	if !ok {
		return ErrMemWrite
	}

	res, err = _alloc.Call(ctx, 8)
	if err != nil {
		return fmt.Errorf("alloc: %w", err)
	}
	sizePtr := res[0]
	defer _free.Call(ctx, sizePtr)

	losslessVal := 0
	if lossless {
		losslessVal = 1
	}

	exactVal := 0
	if exact {
		exactVal = 1
	}

	res, err = _encode.Call(ctx, inPtr, uint64(width), uint64(height), sizePtr, uint64(colorspace), uint64(quality),
		uint64(method), uint64(losslessVal), uint64(exactVal))
	if err != nil {
		return fmt.Errorf("encode: %w", err)
	}
	defer _free.Call(ctx, res[0])

	size, ok := mod.Memory().ReadUint64Le(uint32(sizePtr))
	if !ok {
		return ErrMemRead
	}

	if size == 0 {
		return ErrEncode
	}

	out, ok := mod.Memory().Read(uint32(res[0]), uint32(size))
	if !ok {
		return ErrMemRead
	}

	_, err = w.Write(out)
	if err != nil {
		return fmt.Errorf("write: %w", err)
	}

	return nil
}

var (
	rt wazero.Runtime
	cm wazero.CompiledModule
	mc wazero.ModuleConfig

	initOnce = sync.OnceFunc(initialize)
)

func initialize() {
	ctx := context.Background()
	rt = wazero.NewRuntime(ctx)

	r, err := gzip.NewReader(bytes.NewReader(webpWasm))
	if err != nil {
		panic(err)
	}
	defer r.Close()

	var data bytes.Buffer
	_, err = data.ReadFrom(r)
	if err != nil {
		panic(err)
	}

	cm, err = rt.CompileModule(ctx, data.Bytes())
	if err != nil {
		panic(err)
	}

	wasi_snapshot_preview1.MustInstantiate(ctx, rt)
	mc = wazero.NewModuleConfig().WithStderr(os.Stderr).WithStdout(os.Stdout)
}
