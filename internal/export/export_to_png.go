package export

import (
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"os"

	xdraw "golang.org/x/image/draw"
	"golang.org/x/image/font"
	"golang.org/x/image/font/opentype"
	"golang.org/x/image/math/fixed"

	_ "embed"
)

//go:embed assets/NotoSansMono-VariableFont_wdth,wght.ttf
var Font []byte

type ASCIIExportOptions struct {
	FontSize     int
	DPI          int
	BG           color.Color
	FG           color.Color
	FontTTFPath  string
	TargetAspect float64
	RenderColor  bool
}

func loadExportFontBytes(fontPath string) ([]byte, error) {
	if fontPath == "" {
		return Font, nil
	}

	fontBytes, err := os.ReadFile(fontPath)
	if err != nil {
		return nil, err
	}

	return fontBytes, nil
}

func ASCIIToPNG(runeArray [][]rune, colorArray [][]color.NRGBA, outPath string, opt ASCIIExportOptions) error {
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

	fontBytes, err := loadExportFontBytes(opt.FontTTFPath)
	if err != nil {
		return err
	}

	tt, err := opentype.Parse(fontBytes)
	if err != nil {
		return err
	}

	face, err := opentype.NewFace(tt, &opentype.FaceOptions{
		Size:    float64(opt.FontSize),
		DPI:     float64(opt.DPI),
		Hinting: font.HintingFull,
	})
	if err != nil {
		return err
	}
	defer face.Close()

	d := &font.Drawer{Face: face}

	metrics := face.Metrics()
	lineH := metrics.Height.Round()
	ascent := metrics.Ascent.Ceil()

	rows := len(runeArray)
	cols := 0
	for _, row := range runeArray {
		if len(row) > cols {
			cols = len(row)
		}
	}

	if rows < 1 {
		rows = 1
	}
	if cols < 1 {
		cols = 1
	}

	cellW := d.MeasureString("M").Ceil()
	if cellW < 1 {
		cellW = 1
	}
	if lineH < 1 {
		lineH = 1
	}

	w := cols * cellW
	h := rows * lineH

	img := image.NewRGBA(image.Rect(0, 0, w, h))
	draw.Draw(img, img.Bounds(), &image.Uniform{C: opt.BG}, image.Point{}, draw.Src)

	d.Dst = img
	d.Src = image.NewUniform(opt.FG)

	if !opt.RenderColor {
		for y, row := range runeArray {
			baselineY := y*lineH + ascent
			d.Dot = fixed.P(0, baselineY)
			d.DrawString(string(row))
		}
	} else {
		for y, row := range runeArray {
			baselineY := y*lineH + ascent
			for x, r := range row {
				d.Src = image.NewUniform(colorArray[y][x])
				d.Dot = fixed.P(x*cellW, baselineY)
				d.DrawString(string(r))
			}
		}
	}

	// aspect correction
	if opt.TargetAspect > 0 {
		currentAspect := float64(cellW) / float64(lineH)
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

	f, err := os.Create(outPath)
	if err != nil {
		return err
	}
	defer func() { _ = f.Close() }()

	return png.Encode(f, img)
}
