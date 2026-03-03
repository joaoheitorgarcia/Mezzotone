package export

import (
	"fmt"
	"image"
	"image/color"
	"image/color/palette"
	"image/draw"
	"image/gif"
	"os"
	"runtime"
	"sync"
	"time"

	xdraw "golang.org/x/image/draw"
	"golang.org/x/image/font"
	"golang.org/x/image/font/opentype"
	"golang.org/x/image/math/fixed"
)

type ASCIIGIFFrame struct {
	FrameRunes  [][]rune
	Duration    time.Duration
	FrameColors [][]color.NRGBA
}

func ASCIIFramesToGIF(frames []ASCIIGIFFrame, outPath string, opt ASCIIExportOptions) error {
	if len(frames) == 0 {
		return fmt.Errorf("no frames to export")
	}

	rendered := make([]*image.RGBA, len(frames))
	workers := runtime.GOMAXPROCS(0)
	if workers < 1 {
		workers = 1
	}
	if workers > len(frames) {
		workers = len(frames)
	}

	jobs := make(chan int, len(frames))
	var wg sync.WaitGroup
	var errOnce sync.Once
	var firstErr error

	setErr := func(err error) {
		errOnce.Do(func() {
			firstErr = err
		})
	}

	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()

			r, err := newASCIIRenderer(opt)
			if err != nil {
				setErr(err)
				return
			}
			defer r.Close()

			for frameIdx := range jobs {
				img, err := r.RenderFrame(frames[frameIdx], opt.RenderColor)
				if err != nil {
					setErr(err)
					continue
				}
				rendered[frameIdx] = img
			}
		}()
	}

	for i := range frames {
		jobs <- i
	}
	close(jobs)
	wg.Wait()

	if firstErr != nil {
		return firstErr
	}

	maxW := 1
	maxH := 1
	for _, img := range rendered {
		if img == nil {
			return fmt.Errorf("failed to render one or more gif frames")
		}
		if img.Bounds().Dx() > maxW {
			maxW = img.Bounds().Dx()
		}
		if img.Bounds().Dy() > maxH {
			maxH = img.Bounds().Dy()
		}
	}

	gifFrames := make([]*image.Paletted, 0, len(rendered))
	delays := make([]int, 0, len(rendered))
	for i, img := range rendered {
		canvas := image.NewRGBA(image.Rect(0, 0, maxW, maxH))
		draw.Draw(canvas, canvas.Bounds(), &image.Uniform{C: opt.BG}, image.Point{}, draw.Src)
		draw.Draw(canvas, image.Rect(0, 0, img.Bounds().Dx(), img.Bounds().Dy()), img, image.Point{}, draw.Over)

		paletted := image.NewPaletted(canvas.Bounds(), palette.Plan9)
		draw.FloydSteinberg.Draw(paletted, paletted.Rect, canvas, image.Point{})
		gifFrames = append(gifFrames, paletted)

		delay := int(frames[i].Duration / (10 * time.Millisecond))
		if delay < 1 {
			delay = 1
		}
		delays = append(delays, delay)
	}

	f, err := os.Create(outPath)
	if err != nil {
		return err
	}
	defer func() { _ = f.Close() }()

	return gif.EncodeAll(f, &gif.GIF{
		Image: gifFrames,
		Delay: delays,
	})
}

type asciiRenderer struct {
	opt  ASCIIExportOptions
	face font.Face
}

func newASCIIRenderer(opt ASCIIExportOptions) (*asciiRenderer, error) {
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
		return nil, err
	}

	tt, err := opentype.Parse(fontBytes)
	if err != nil {
		return nil, err
	}

	face, err := opentype.NewFace(tt, &opentype.FaceOptions{
		Size:    float64(opt.FontSize),
		DPI:     float64(opt.DPI),
		Hinting: font.HintingFull,
	})
	if err != nil {
		return nil, err
	}

	return &asciiRenderer{
		opt:  opt,
		face: face,
	}, nil
}

func (r *asciiRenderer) Close() {
	if r.face != nil {
		_ = r.face.Close()
	}
}

func (r *asciiRenderer) RenderFrame(frame ASCIIGIFFrame, renderColor bool) (*image.RGBA, error) {
	d := &font.Drawer{Face: r.face}

	metrics := r.face.Metrics()
	lineH := metrics.Height.Round()
	ascent := metrics.Ascent.Ceil()

	rows := len(frame.FrameRunes)
	cols := 0
	for _, row := range frame.FrameRunes {
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
	draw.Draw(img, img.Bounds(), &image.Uniform{C: r.opt.BG}, image.Point{}, draw.Src)

	d = &font.Drawer{
		Dst:  img,
		Src:  image.NewUniform(r.opt.FG),
		Face: r.face,
	}

	if !renderColor {
		for y, row := range frame.FrameRunes {
			baselineY := y*lineH + ascent
			d.Dot = fixed.P(0, baselineY)
			d.DrawString(string(row))
		}
	} else {
		for y, row := range frame.FrameRunes {
			baselineY := y*lineH + ascent
			for x, r := range row {
				d.Src = image.NewUniform(frame.FrameColors[y][x])
				d.Dot = fixed.P(x*cellW, baselineY)
				d.DrawString(string(r))
			}
		}
	}

	if r.opt.TargetAspect > 0 {
		currentAspect := float64(cellW) / float64(lineH)
		scaleX := r.opt.TargetAspect / currentAspect

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

	return img, nil
}
