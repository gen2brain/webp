package webp

import (
	"fmt"
	"image"
	"image/color"
	"io"
	"unsafe"
)

func decode(r io.Reader, configOnly, decodeAll bool) (*WEBP, image.Config, error) {
	var cfg image.Config
	var data []byte
	var err error

	mod := New()
	mod.X_initialize()

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

	inPtr := mod.Xmalloc(int32(inSize))
	defer mod.Xfree(inPtr)

	ok := mod.write(inPtr, data)
	if !ok {
		return nil, cfg, ErrMemWrite
	}

	ptr := mod.Xmalloc(4 * 4)
	defer mod.Xfree(ptr)

	widthPtr := ptr
	heightPtr := ptr + 4
	countPtr := ptr + 8
	animPtr := ptr + 12

	all := int32(0)
	if decodeAll {
		all = 1
	}

	res := mod.Xdecode(inPtr, int32(inSize), 1, all, widthPtr, heightPtr, countPtr, animPtr, 0, 0)
	if res == 0 {
		return nil, cfg, ErrDecode
	}

	width, ok := mod.readUint32(widthPtr)
	if !ok {
		return nil, cfg, ErrMemRead
	}

	height, ok := mod.readUint32(heightPtr)
	if !ok {
		return nil, cfg, ErrMemRead
	}

	anim, ok := mod.readUint32(animPtr)
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
		count, ok := mod.readUint32(countPtr)
		if !ok {
			return nil, cfg, ErrMemRead
		}

		size := cfg.Width * cfg.Height * 4

		outPtr := mod.Xmalloc(int32(size * int(count)))
		defer mod.Xfree(outPtr)

		delayPtr := mod.Xmalloc(int32(4 * int(count)))
		defer mod.Xfree(delayPtr)

		res = mod.Xdecode(inPtr, int32(inSize), 0, all, widthPtr, heightPtr, countPtr, animPtr, delayPtr, outPtr)
		if res == 0 {
			return nil, cfg, ErrDecode
		}

		for i := 0; i < int(count); i++ {
			out, ok := mod.read(outPtr+int32(i*size), int32(size))
			if !ok {
				return nil, cfg, ErrMemRead
			}

			img := image.NewRGBA(image.Rect(0, 0, cfg.Width, cfg.Height))
			img.Pix = out

			images = append(images, img)

			d, ok := mod.readUint32(delayPtr + int32(i*4))
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

	outPtr := mod.Xmalloc(int32(size))
	defer mod.Xfree(outPtr)

	res = mod.Xdecode(inPtr, int32(inSize), 0, all, widthPtr, heightPtr, countPtr, animPtr, 0, outPtr)
	if res == 0 {
		return nil, cfg, ErrDecode
	}

	out, ok := mod.read(outPtr, int32(size))
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
	mod := New()
	mod.X_initialize()

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

	inPtr := mod.Xmalloc(int32(len(data)))
	defer mod.Xfree(inPtr)

	ok := mod.write(inPtr, data)
	if !ok {
		return ErrMemWrite
	}

	sizePtr := mod.Xmalloc(8)
	defer mod.Xfree(sizePtr)

	losslessVal := int32(0)
	if lossless {
		losslessVal = 1
	}

	exactVal := int32(0)
	if exact {
		exactVal = 1
	}

	outPtr := mod.Xencode(inPtr, int32(width), int32(height), sizePtr, int32(colorspace), int32(quality),
		int32(method), losslessVal, exactVal)
	defer mod.Xfree(outPtr)

	size, ok := mod.readUint64(sizePtr)
	if !ok {
		return ErrMemRead
	}

	if size == 0 {
		return ErrEncode
	}

	out, ok := mod.read(outPtr, int32(size))
	if !ok {
		return ErrMemRead
	}

	_, err := w.Write(out)
	if err != nil {
		return fmt.Errorf("write: %w", err)
	}

	return nil
}

func (m *Module) write(ptr int32, data []byte) bool {
	if ptr < 0 || int(ptr)+len(data) > len(m.memory) {
		return false
	}

	copy(m.memory[ptr:], data)

	return true
}

func (m *Module) read(ptr, size int32) ([]byte, bool) {
	if ptr < 0 || size < 0 || int(ptr)+int(size) > len(m.memory) {
		return nil, false
	}

	return m.memory[ptr : ptr+size : ptr+size], true
}

func (m *Module) readUint32(ptr int32) (uint32, bool) {
	if ptr < 0 || int(ptr)+4 > len(m.memory) {
		return 0, false
	}

	return load32(m.memory[ptr:]), true
}

func (m *Module) readUint64(ptr int32) (uint64, bool) {
	if ptr < 0 || int(ptr)+8 > len(m.memory) {
		return 0, false
	}

	return load64(m.memory[ptr:]), true
}
