// Command mkicon generates the gmid app icon: an amber rounded square with a
// dark Inter Bold "g", matching the in-app top-left logo.
//
// Run from the repository root:
//
//	go run ./tools/mkicon
package main

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"

	"github.com/fogleman/gg"
	xdraw "golang.org/x/image/draw"
	"golang.org/x/image/font"
)

const (
	sourceSize = 1024
	corner     = 230
	fontSize   = 620
)

var (
	orange = color.RGBA{0xe8, 0xa0, 0x27, 0xff}
	ink    = color.RGBA{0x18, 0x10, 0x05, 0xff}
)

func main() {
	img, err := render()
	if err != nil {
		fmt.Fprintln(os.Stderr, "mkicon:", err)
		os.Exit(1)
	}
	out := "assets/icons"
	if err := os.MkdirAll(out, 0o755); err != nil {
		fmt.Fprintln(os.Stderr, "mkicon:", err)
		os.Exit(1)
	}
	sizes := []int{1024, 512, 256, 128, 64, 48, 32, 16}
	bySize := map[int][]byte{}
	for _, size := range sizes {
		data, err := encodePNG(resize(img, size))
		if err != nil {
			fmt.Fprintln(os.Stderr, "mkicon:", err)
			os.Exit(1)
		}
		bySize[size] = data
		path := filepath.Join(out, fmt.Sprintf("gmid-%d.png", size))
		if err := os.WriteFile(path, data, 0o644); err != nil {
			fmt.Fprintln(os.Stderr, "mkicon:", err)
			os.Exit(1)
		}
		fmt.Println(path)
	}
	icns, err := encodeICNS(bySize)
	if err != nil {
		fmt.Fprintln(os.Stderr, "mkicon:", err)
		os.Exit(1)
	}
	path := filepath.Join(out, "gmid.icns")
	if err := os.WriteFile(path, icns, 0o644); err != nil {
		fmt.Fprintln(os.Stderr, "mkicon:", err)
		os.Exit(1)
	}
	fmt.Println(path)

	// Canonical icon name used by the Linux desktop entry (`Icon=gmid`).
	path = filepath.Join(out, "gmid.png")
	if err := os.WriteFile(path, bySize[256], 0o644); err != nil {
		fmt.Fprintln(os.Stderr, "mkicon:", err)
		os.Exit(1)
	}
	fmt.Println(path)
}

func render() (*image.RGBA, error) {
	face, err := gg.LoadFontFace("assets/fonts/Inter-Bold.ttf", fontSize)
	if err != nil {
		return nil, err
	}
	dc := gg.NewContext(sourceSize, sourceSize)
	dc.SetColor(orange)
	dc.DrawRoundedRectangle(0, 0, sourceSize, sourceSize, corner)
	dc.Fill()

	dc.SetFontFace(face)
	dc.SetColor(ink)
	bounds, _ := font.BoundString(face, "g")
	minX := float64(bounds.Min.X) / 64
	maxX := float64(bounds.Max.X) / 64
	minY := float64(bounds.Min.Y) / 64
	maxY := float64(bounds.Max.Y) / 64
	x := (sourceSize-(maxX-minX))/2 - minX
	baseline := sourceSize/2 - (minY+maxY)/2
	dc.DrawString("g", x, baseline)
	return dc.Image().(*image.RGBA), nil
}

func resize(src *image.RGBA, size int) *image.RGBA {
	dst := image.NewRGBA(image.Rect(0, 0, size, size))
	xdraw.CatmullRom.Scale(dst, dst.Bounds(), src, src.Bounds(), xdraw.Src, nil)
	return dst
}

func encodePNG(img image.Image) ([]byte, error) {
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// encodeICNS writes a modern PNG-based ICNS container. macOS reads the ic07..
// ic14 entries; icp4..icp6 cover legacy small sizes.
func encodeICNS(pngs map[int][]byte) ([]byte, error) {
	entries := []struct {
		typ  string
		size int
	}{
		{"icp4", 16},
		{"icp5", 32},
		{"icp6", 64},
		{"ic07", 128},
		{"ic08", 256},
		{"ic09", 512},
		{"ic10", 1024},
		{"ic11", 32},
		{"ic12", 64},
		{"ic13", 256},
		{"ic14", 512},
	}
	var body bytes.Buffer
	for _, e := range entries {
		data := pngs[e.size]
		if len(data) == 0 {
			return nil, fmt.Errorf("missing icon size %d", e.size)
		}
		body.WriteString(e.typ)
		_ = binary.Write(&body, binary.BigEndian, uint32(len(data)+8))
		body.Write(data)
	}
	var out bytes.Buffer
	out.WriteString("icns")
	_ = binary.Write(&out, binary.BigEndian, uint32(body.Len()+8))
	out.Write(body.Bytes())
	return out.Bytes(), nil
}
