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

	gifFrames := make([]*image.Paletted, len(frames))
	delays := make([]int, len(frames))
	renderer, err := newASCIIRenderer(opt)
	if err != nil {
		return err
	}
	defer renderer.Close()

	maxRows := 1
	maxCols := 1
	for _, frame := range frames {
		if len(frame.FrameRunes) > maxRows {
			maxRows = len(frame.FrameColors)
		}
		for _, frameColor := range frame.FrameRunes {
			if len(frameColor) > maxCols {
				maxCols = len(frameColor)
			}
		}
	}

	d := &font.Drawer{Face: renderer.face}

	metrics := renderer.face.Metrics()
	lineH := metrics.Height.Round()
	cellW := d.MeasureString("M").Ceil()

	if cellW < 1 {
		cellW = 1
	}
	if lineH < 1 {
		lineH = 1
	}

	fontVars := fontVariables{
		width:  maxCols * cellW,
		height: maxRows * lineH,
		ascent: metrics.Ascent.Ceil(),
		lineH:  lineH,
		cellW:  cellW,
	}

	workers := min(4, runtime.GOMAXPROCS(0), len(frames))
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
				img, err := r.RenderFrame(frames[frameIdx], opt.RenderColor, fontVars)
				if err != nil {
					setErr(err)
					continue
				}

				canvas := image.NewRGBA(image.Rect(0, 0, img.Bounds().Dx(), img.Bounds().Dy()))
				draw.Draw(canvas, canvas.Bounds(), &image.Uniform{C: opt.BG}, image.Point{}, draw.Src)
				draw.Draw(canvas, image.Rect(0, 0, img.Bounds().Dx(), img.Bounds().Dy()), img, image.Point{}, draw.Over)

				paletted := image.NewPaletted(canvas.Bounds(), palette.Plan9)
				draw.Draw(paletted, image.Rect(0, 0, img.Bounds().Dx(), img.Bounds().Dy()), canvas, image.Point{}, draw.Over)
				gifFrames[frameIdx] = paletted

				delay := int(frames[frameIdx].Duration / (10 * time.Millisecond))
				if delay < 1 {
					delay = 1
				}
				delays[frameIdx] = delay
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

	//reduce gif size
	previousFrame := gifFrames[0]
	for i, frame := range gifFrames {
		if i == 0 {
			continue
		}
		diffBox := getDiffBounds(previousFrame, frame)
		if diffBox.Empty() {
			gifFrames[i] = image.NewPaletted(image.Rect(0, 0, 1, 1), frame.Palette)
			continue
		}
		if diffBox.Dx()*diffBox.Dy() > frame.Rect.Dx()*frame.Rect.Dy() {
			continue
		}

		croppedPallete := image.NewPaletted(diffBox, frame.Palette)
		r := croppedPallete.Rect
		for y := r.Min.Y; y < r.Max.Y; y++ {
			srcOff := (y-frame.Rect.Min.Y)*frame.Stride + (r.Min.X - frame.Rect.Min.X)
			dstOff := (y-croppedPallete.Rect.Min.Y)*croppedPallete.Stride + (r.Min.X - croppedPallete.Rect.Min.X)
			copy(croppedPallete.Pix[dstOff:dstOff+r.Dx()], frame.Pix[srcOff:srcOff+r.Dx()])
		}
		gifFrames[i] = croppedPallete
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

func getDiffBounds(previous, current *image.Paletted) image.Rectangle {
	rec := previous.Rect
	minX, maxX, minY, maxY := -1, -1, -1, -1

	for y := rec.Min.Y; y < rec.Max.Y; y++ {
		previousRow := (y - rec.Min.Y) * previous.Stride
		currentRow := (y - rec.Min.Y) * previous.Stride
		for x := rec.Min.X; x < rec.Max.X; x++ {
			i := x - rec.Min.X
			if previous.Pix[previousRow+i] != current.Pix[currentRow+i] {
				if minX == -1 {
					minX, maxX = x, x
					minY, maxY = y, y
				} else {
					if x < minX {
						minX = x
					}
					if x > maxX {
						maxX = x
					}
					if y < minY {
						minY = y
					}
					if y > maxY {
						maxY = y
					}
				}
			}
		}
	}
	if maxX == -1 {
		return image.Rectangle{}
	}
	return image.Rect(minX, minY, maxX+1, maxY+1)
}

type asciiRenderer struct {
	opt  ASCIIExportOptions
	face font.Face
}

type fontVariables struct {
	width  int
	height int
	lineH  int
	ascent int
	cellW  int
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

func (r *asciiRenderer) RenderFrame(frame ASCIIGIFFrame, renderColor bool, fontVars fontVariables) (*image.RGBA, error) {
	img := image.NewRGBA(image.Rect(0, 0, fontVars.width, fontVars.height))
	draw.Draw(img, img.Bounds(), &image.Uniform{C: r.opt.BG}, image.Point{}, draw.Src)

	d := &font.Drawer{
		Dst:  img,
		Src:  image.NewUniform(r.opt.FG),
		Face: r.face,
	}

	if !renderColor {
		for y, row := range frame.FrameRunes {
			baselineY := y*fontVars.lineH + fontVars.ascent
			d.Dot = fixed.P(0, baselineY)
			d.DrawString(string(row))
		}
	} else {
		for y, row := range frame.FrameRunes {
			baselineY := y*fontVars.lineH + fontVars.ascent
			for x, r := range row {
				d.Src = image.NewUniform(frame.FrameColors[y][x])
				d.Dot = fixed.P(x*fontVars.cellW, baselineY)
				d.DrawString(string(r))
			}
		}
	}

	return img, nil
}
