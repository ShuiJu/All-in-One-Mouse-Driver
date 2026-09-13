package main

import (
	"bytes"
	"embed"
	"image"
	"image/draw"
	"image/png"
	"math"
	"mousehub/internal/hub"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"unsafe"
)

//go:embed assets/app.ico assets/mice/*.png
var embeddedAssets embed.FS

var msimg32 = syscall.NewLazyDLL("msimg32.dll")

type rgbaBitmap struct {
	w, h   int
	handle uintptr
}

var (
	mouseBitmaps = map[string]*rgbaBitmap{}
	logoIcon     uintptr
	logoIconPath string
)

func loadPNGAsset(name string) (*rgbaBitmap, error) {
	raw, err := embeddedAssets.ReadFile("assets/mice/" + name + ".png")
	if err != nil {
		return nil, err
	}
	src, err := png.Decode(bytes.NewReader(raw))
	if err != nil {
		return nil, err
	}
	b := src.Bounds()
	w, h := b.Dx(), b.Dy()
	rgba := image.NewRGBA(image.Rect(0, 0, w, h))
	draw.Draw(rgba, rgba.Bounds(), src, b.Min, draw.Src)
	// GDI AlphaBlend wants premultiplied BGRA top-down DIB.
	px := rgba.Pix
	for i := 0; i+3 < len(px); i += 4 {
		a := px[i+3]
		px[i] = uint8(int(px[i]) * int(a) / 255)
		px[i+1] = uint8(int(px[i+1]) * int(a) / 255)
		px[i+2] = uint8(int(px[i+2]) * int(a) / 255)
		px[i], px[i+2] = px[i+2], px[i]
	}
	type bitmapInfoHeader struct {
		Size                                   uint32
		Width, Height                          int32
		Planes, BitCount                       uint16
		Compression, SizeImage                 uint32
		XPelsPerMeter, YPelsPerMeter           int32
		ClrUsed, ClrImportant                  uint32
	}
	var bi struct {
		Header bitmapInfoHeader
	}
	bi.Header = bitmapInfoHeader{Size: 40, Width: int32(w), Height: -int32(h), Planes: 1, BitCount: 32, SizeImage: uint32(w * h * 4)}
	var bits uintptr
	screen, _, _ := user32.NewProc("GetDC").Call(0)
	hbmp, _, errCall := gdi.NewProc("CreateDIBSection").Call(screen, uintptr(unsafe.Pointer(&bi)), 0, uintptr(unsafe.Pointer(&bits)), 0, 0)
	user32.NewProc("ReleaseDC").Call(0, screen)
	if hbmp == 0 || bits == 0 {
		return nil, errCall
	}
	dst := unsafe.Slice((*byte)(unsafe.Pointer(bits)), w*h*4)
	copy(dst, px)
	return &rgbaBitmap{w: w, h: h, handle: hbmp}, nil
}

func loadEmbeddedAssets() {
	for _, key := range []string{"lamzu-inca", "vaxee-e1", "vaxee-np01sv2"} {
		if bmp, err := loadPNGAsset(key); err == nil {
			mouseBitmaps[key] = bmp
		}
	}
	raw, err := embeddedAssets.ReadFile("assets/app.ico")
	if err != nil {
		return
	}
	dir := os.TempDir()
	path := filepath.Join(dir, "MouseHub-app.ico")
	if old, err := os.ReadFile(path); err != nil || !bytes.Equal(old, raw) {
		_ = os.WriteFile(path, raw, 0600)
	}
	logoIconPath = path
	h, _, _ := user32.NewProc("LoadImageW").Call(0, uintptr(unsafe.Pointer(utf(path))), 1, 0, 0, 0x00000010)
	logoIcon = h
}

func releaseEmbeddedAssets() {
	for _, b := range mouseBitmaps {
		if b != nil && b.handle != 0 {
			gdi.NewProc("DeleteObject").Call(b.handle)
		}
	}
	mouseBitmaps = map[string]*rgbaBitmap{}
	if logoIcon != 0 {
		user32.NewProc("DestroyIcon").Call(logoIcon)
		logoIcon = 0
	}
}

func mouseAssetKey(d *hub.Device) string {
	if d == nil {
		return ""
	}
	if d.Brand == "lamzu" {
		return "lamzu-inca"
	}
	switch d.MousePID {
	case 0x1010:
		return "vaxee-e1"
	case 0x1011:
		return "vaxee-np01sv2"
	}
	name := strings.ToUpper(d.Name)
	if strings.Contains(name, "NP-01") || strings.Contains(name, "NP01") {
		return "vaxee-np01sv2"
	}
	if strings.Contains(name, "E1") {
		return "vaxee-e1"
	}
	return "vaxee-e1"
}

func (u *nativeUI) drawMouseImage(dc uintptr, key string, x, y, w, h int) bool {
	bmp := mouseBitmaps[key]
	if bmp == nil || bmp.handle == 0 || bmp.w < 1 || bmp.h < 1 || w < 1 || h < 1 {
		return false
	}
	// Aspect-fit inside the target box.
	scale := math.Min(float64(w)/float64(bmp.w), float64(h)/float64(bmp.h))
	dw := int(float64(bmp.w) * scale)
	dh := int(float64(bmp.h) * scale)
	dx := x + (w-dw)/2
	dy := y + (h-dh)/2
	mem, _, _ := gdi.NewProc("CreateCompatibleDC").Call(dc)
	if mem == 0 {
		return false
	}
	old, _, _ := gdi.NewProc("SelectObject").Call(mem, bmp.handle)
	type blend struct {
		Op, Flags           byte
		ConstantAlpha, Mode byte
	}
	fn := blend{Op: 0, Flags: 0, ConstantAlpha: 255, Mode: 1}
	msimg32.NewProc("AlphaBlend").Call(
		dc, uintptr(dx), uintptr(dy), uintptr(dw), uintptr(dh),
		mem, 0, 0, uintptr(bmp.w), uintptr(bmp.h),
		uintptr(*(*uint32)(unsafe.Pointer(&fn))),
	)
	gdi.NewProc("SelectObject").Call(mem, old)
	gdi.NewProc("DeleteDC").Call(mem)
	return true
}
