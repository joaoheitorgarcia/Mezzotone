package export

import (
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"os"
	"strings"

	xdraw "golang.org/x/image/draw"
	"golang.org/x/image/font"
	"golang.org/x/image/font/opentype"
	"golang.org/x/image/math/fixed"

	_ "embed"
)

//go:embed assets/NotoSansMono-VariableFont_wdth,wght.ttf
var Font []byte

type ASCIIExportOptions struct {
	FontSize     float64
	DPI          float64
	BG           color.Color
	FG           color.Color
	TargetAspect float64
}

func ASCIIToPNG(ascii string, outPath string, opt ASCIIExportOptions) error {
	if opt.DPI <= 0 {
		opt.DPI = 72
	}
	if opt.FontSize <= 0 {
		opt.FontSize = 14
	}
	if opt.BG == nil {
		opt.BG = color.Black
	}
	if opt.FG == nil {
		opt.FG = color.White
	}

	tt, err := opentype.Parse(Font)
	if err != nil {
		return err
	}

	face, err := opentype.NewFace(tt, &opentype.FaceOptions{
		Size:    opt.FontSize,
		DPI:     opt.DPI,
		Hinting: font.HintingFull,
	})
	if err != nil {
		return err
	}
	defer face.Close()

	lines := strings.Split(strings.ReplaceAll(ascii, "\r\n", "\n"), "\n")

	d := &font.Drawer{Face: face}

	maxW := 0
	for _, line := range lines {
		w := d.MeasureString(line).Round()
		if w > maxW {
			maxW = w
		}
	}

	metrics := face.Metrics()
	lineH := metrics.Height.Round()
	ascent := metrics.Ascent.Round()

	w := maxW
	h := lineH * len(lines)
	if w < 1 {
		w = 1
	}
	if h < 1 {
		h = 1
	}

	img := image.NewRGBA(image.Rect(0, 0, w, h))
	draw.Draw(img, img.Bounds(), &image.Uniform{C: opt.BG}, image.Point{}, draw.Src)

	d = &font.Drawer{
		Dst:  img,
		Src:  image.NewUniform(opt.FG),
		Face: face,
	}

	y := ascent
	for _, line := range lines {
		d.Dot = fixed.P(0, y)
		d.DrawString(line)
		y += lineH
	}

	// aspect correction
	if opt.TargetAspect > 0 {
		advM := d.MeasureString("M").Round()
		if advM > 0 && lineH > 0 {
			currentAspect := float64(advM) / float64(lineH)
			scaleX := opt.TargetAspect / currentAspect

			if scaleX > 0.01 && scaleX < 100 {
				newW := int(float64(img.Bounds().Dx()) * scaleX)
				if newW < 1 {
					newW = 1
				}
				scaled := image.NewRGBA(image.Rect(0, 0, newW, img.Bounds().Dy()))
				xdraw.ApproxBiLinear.Scale(scaled, scaled.Bounds(), img, img.Bounds(), draw.Over, nil)
				img = scaled
			}
		}
	}

	f, err := os.Create(outPath)
	if err != nil {
		return err
	}
	defer func() { _ = f.Close() }()

	return png.Encode(f, img)
}
