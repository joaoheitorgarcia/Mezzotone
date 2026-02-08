package services

import (
	"fmt"
	"slices"

	"image"
	"image/color"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"math"
	"os"

	_ "golang.org/x/image/bmp"
	_ "golang.org/x/image/tiff"
	_ "golang.org/x/image/webp"
)

// edgeInfo Struct to store edge info from Sobel filter
type edgeInfo struct {
	Magnitude float64
	Angle     float64
}

type RenderOptions struct {
	// textSize: roughly controls how many pixels map to one character horizontally.
	textSize int
	// fontAspect: terminal characters are typically taller than they are wide, vertical cell size is textSize * fontAspect.
	fontAspect float64
	// directionalRender: optional Edge Awareness. Derive edge magnitude/orientation from luminanceGrid and choose glyphs accordingly.
	directionalRender bool
	edgeThreshold     float64
	// reverseChars: invert ramp direction (useful for dark terminals / preference).
	reverseChars bool
	// highContrast: optional contrast curve applied after cell luminance averaging.
	highContrast bool
	runeMode     string
}

func NewRenderOptions(
	textSize int,
	fontAspect float64,
	directionalRender bool,
	edgeThreshold float64,
	reverseChars bool,
	highContrast bool,
	runeMode string,
) (RenderOptions, error) {
	availableRuneMode := []string{"ASCII", "UNICODE", "DOTS", "RECTANGLES", "BARS", "LOADING"}
	if !slices.Contains(availableRuneMode, runeMode) {
		return RenderOptions{}, fmt.Errorf("invalid rune mode: %s", runeMode)
	}

	return RenderOptions{
		textSize:          textSize,
		fontAspect:        fontAspect,
		directionalRender: directionalRender,
		edgeThreshold:     edgeThreshold,
		reverseChars:      reverseChars,
		highContrast:      highContrast,
		runeMode:          runeMode,
	}, nil
}

// Dark to Bright
const asciiRampDarkToBrightStr = "$@B%8&WM#*oahkbdpqwmZO0QLCJUYXzcvunxrjtf()1{}[]?_+~<>i!lI;:,^`. "
const unicodeRampDarkToBrightStr = "█▓▒░■□@&%$#*+=~:;!,\".^`' "
const dotsRampDarkToBrightStr = "●∙•· "
const rectanglesRampDarkToBrightStr = "█▓▒░ "
const barsRampDarkToBrightStr = "█▇▆▅▄▃▂▁ "
const loadingRampDarkToBrightStr = "⣿⣷⣧⣇⣆⣄⣀ "

// Bright to Dark
const asciiRampBrightToDarkStr = " .`^,:;Il!i><~+_?][}{1)(ftjrxnuvczXYUJCLQ0OZmwqpdbkhao*#MW&8%B@$"
const unicodeRampBrightToDarkStr = " '`^.\",!;:~=+*#$%&@□■░▒▓█"
const dotsRampBrightToDarkStr = " ·•∙●"
const rectanglesRampBrightToDarkStr = " ░▒▓█"
const barsRampBrightToDarkStr = " ▁▂▃▄▅▆▇█"
const loadingRampBrightToDarkStr = " ⣀⣄⣆⣇⣧⣷⣿"

func ConvertImageToString(filePath string, renderOptions RenderOptions) ([][]rune, error) {
	var outputChars [][]rune

	f, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer func() { _ = f.Close() }()

	_ = Logger().Info(fmt.Sprintf("Successfully Loaded: %s", filePath))

	inputImg, format, err := image.Decode(f)
	if err != nil {
		return nil, err
	}
	_ = Logger().Info(fmt.Sprintf("format: %s", format))

	// Compute grid resolution (cols x rows) based on image size + character cell size.
	cols, rows := getColsAndRows(inputImg, renderOptions.textSize, renderOptions.fontAspect)
	cellWidth := float64(inputImg.Bounds().Dx()) / float64(cols)
	cellHeight := float64(inputImg.Bounds().Dy()) / float64(rows)
	if cellWidth <= 0 {
		cellWidth = 1
	}
	if cellHeight <= 0 {
		cellHeight = 1
	}

	outputChars = make([][]rune, rows)
	for r := 0; r < rows; r++ {
		outputChars[r] = make([]rune, cols)
	}

	// Build a luminance grid (rows x cols) where each cell is 0..1.
	// Each cell luminance is computed by averaging pixels in the corresponding image region.
	luminanceGrid, err := buildLuminanceGrid(inputImg, cols, rows, renderOptions.highContrast)
	if err != nil {
		return nil, err
	}
	_ = Logger().Info(fmt.Sprintf("Successfully Build LumaGrid for %s", filePath))

	edgeThreshold := 0.0
	edgeInfos := make([][]edgeInfo, 0)
	if renderOptions.directionalRender {
		edgeThresholdPercentile := clamp01(renderOptions.edgeThreshold)
		edgeThreshold = edgeThresholdPercentile

		dogGrid := differenceOfGaussiansGrid(luminanceGrid, 0.5, 1.0)
		edgeInfos = applySobelFilter(dogGrid, cellWidth, cellHeight)
	}

	_ = Logger().Info(fmt.Sprintf("Beginning image conversion"))

	// Convert each luminance cell to a glyph using the chosen ramp.
	// indices are [row][col] matching outputChars.
	for i := 0; i < len(luminanceGrid); i++ {
		for j := 0; j < len(luminanceGrid[i]); j++ {
			//if directionalRender true and Magnitude surpasses threshold replace with directional char
			if renderOptions.directionalRender && edgeInfos[i][j].Magnitude > edgeThreshold {
				outputChars[i][j] = getEdgeRuneFromGradient(edgeInfos[i][j], renderOptions.runeMode)
				if outputChars[i][j] == ' ' {
					outputChars[i][j] = getRuneForLuminanceValue(luminanceGrid[i][j], renderOptions.runeMode, renderOptions.reverseChars)
				}
			} else {
				outputChars[i][j] = getRuneForLuminanceValue(luminanceGrid[i][j], renderOptions.runeMode, renderOptions.reverseChars)
			}
		}
	}

	_ = Logger().Info(fmt.Sprintf("Finished image conversion"))
	return outputChars, nil
}

func ImageRuneArrayIntoString(runeArray [][]rune) string {
	outpurString := ""
	for x := 0; x < len(runeArray); x++ {
		for y := 0; y < len(runeArray[x]); y++ {
			outpurString += string(runeArray[x][y])
		}
		outpurString += "\n"
	}

	return outpurString
}

// Calculates Columns and Rows for given TextSize and FontAspect
func getColsAndRows(img image.Image, textSize int, fontAspect float64) (cols, rows int) {
	b := img.Bounds()
	imgW, imgH := b.Dx(), b.Dy()

	charW := textSize
	charH := int(float64(textSize) * fontAspect)
	if charW <= 0 {
		charW = 8
	}
	if charH <= 0 {
		charH = 16
	}

	cols = (imgW + charW - 1) / charW
	rows = (imgH + charH - 1) / charH

	if cols < 1 {
		cols = 1
	}
	if rows < 1 {
		rows = 1
	}

	return cols, rows
}

// Builds a grid of averaged luminance values in [0..1].
func buildLuminanceGrid(inputImg image.Image, cols, rows int, highContrast bool) ([][]float64, error) {

	imgBounds := inputImg.Bounds()
	imgWidth, imgHeight := imgBounds.Dx(), imgBounds.Dy()

	cellWidth := imgWidth / cols
	cellHeight := imgHeight / rows

	// Safety fallback if cols/rows are weird (should be prevented earlier).
	if cellWidth <= 0 {
		cellWidth = 8
	}
	if cellHeight <= 0 {
		cellHeight = 16
	}

	// Allocate luminance grid.
	grid := make([][]float64, rows)
	for gridRow := 0; gridRow < rows; gridRow++ {
		grid[gridRow] = make([]float64, cols)
	}

	for gridRow := 0; gridRow < rows; gridRow++ {
		// Pixel Y-range for this grid row.
		cellRowPixelStartY := gridRow * cellHeight
		cellRowPixelEndY := cellRowPixelStartY + cellHeight
		if cellRowPixelStartY >= imgHeight {
			cellRowPixelStartY = imgHeight
		}
		if cellRowPixelEndY > imgHeight {
			cellRowPixelEndY = imgHeight
		}

		for gridCol := 0; gridCol < cols; gridCol++ {
			// Pixel X-range for this grid column.
			cellColPixelStartX := gridCol * cellWidth
			cellColPixelEndX := cellColPixelStartX + cellWidth
			if cellColPixelStartX >= imgWidth {
				cellColPixelStartX = imgWidth
			}
			if cellColPixelEndX > imgWidth {
				cellColPixelEndX = imgWidth
			}

			// Fallback guard (should not happen if dimensions are sane).
			if cellColPixelEndX <= cellColPixelStartX || cellRowPixelEndY <= cellRowPixelStartY {
				grid[gridRow][gridCol] = 0
				continue
			}

			var lumaSum float64
			var sampleCount float64

			for y := cellRowPixelStartY; y < cellRowPixelEndY; y++ {
				for x := cellColPixelStartX; x < cellColPixelEndX; x++ {
					c := color.NRGBAModel.Convert(
						inputImg.At(imgBounds.Min.X+x, imgBounds.Min.Y+y),
					).(color.NRGBA)

					// Skip mostly transparent pixels to prevent background bleed.
					if c.A < 10 {
						continue
					}

					// Luminance is computed as 0..1.
					pixelLuminance := calculateLuminance(c.R, c.G, c.B)
					lumaSum += pixelLuminance
					sampleCount++
				}
			}

			// Average luminance;
			// if all transparent, treat as black.
			var cellLuma float64
			if sampleCount == 0 {
				cellLuma = 0
			} else {
				cellLuma = lumaSum / sampleCount
			}

			// Optional contrast remap
			if highContrast {
				cellLuma = applyContrast(cellLuma, 1.7)
			}

			grid[gridRow][gridCol] = clamp01(cellLuma)
		}
	}

	return grid, nil
}

/*
Calculates luminance from rgb values and normalizes them from 0..255 into 0..1

	Uses standard relative luminance weights (Rec.709 / sRGB), where green contributes the most to perceived brightness
*/
func calculateLuminance(red uint8, green uint8, blue uint8) float64 {
	luminance := 0.2126*float64(red) + 0.7152*float64(green) + 0.0722*float64(blue)
	return luminance / 255.0
}

// Get the rune correspondent to luminance in selected ramp
func getRuneForLuminanceValue(luminance float64, runeMode string, reverseChars bool) rune {
	var ramp []rune

	switch runeMode {
	case "UNICODE":
		if reverseChars {
			ramp = []rune(unicodeRampBrightToDarkStr)
		} else {
			ramp = []rune(unicodeRampDarkToBrightStr)
		}
	case "DOTS":
		if reverseChars {
			ramp = []rune(dotsRampBrightToDarkStr)
		} else {
			ramp = []rune(dotsRampDarkToBrightStr)
		}
	case "RECTANGLES":
		if reverseChars {
			ramp = []rune(rectanglesRampBrightToDarkStr)
		} else {
			ramp = []rune(rectanglesRampDarkToBrightStr)
		}
	case "BARS":
		if reverseChars {
			ramp = []rune(barsRampBrightToDarkStr)
		} else {
			ramp = []rune(barsRampDarkToBrightStr)
		}
	case "LOADING":
		if reverseChars {
			ramp = []rune(loadingRampBrightToDarkStr)
		} else {
			ramp = []rune(loadingRampDarkToBrightStr)
		}
	default:
	case "ASCII":
		if reverseChars {
			ramp = []rune(asciiRampBrightToDarkStr)
		} else {
			ramp = []rune(asciiRampDarkToBrightStr)
		}

	}

	// Map luminance to an index in the ramp:
	index := int(luminance * float64(len(ramp)-1))

	_ = Logger().Info(
		fmt.Sprintf(
			"brightness: %.2f | character: %s | character index: %d",
			luminance, string(ramp[index]), index,
		),
	)

	return ramp[index]
}

// Clamp to [0..1] to keep mapping stable.
func clamp01(x float64) float64 {

	if x < 0 {
		return 0
	}
	if x > 1 {
		return 1
	}
	return x
}

// Applies contrast to lumiance levels with contrast curve at 0.5
func applyContrast(l float64, factor float64) float64 {
	return clamp01((l-0.5)*factor + 0.5)
}

/*
Applies Sobel filter to lumaGrid

	Searches for biggest Change in luminance in adjacent grid values and calculates magnitude and angle of the change
	Returns edgeInfo grid with normalized values

Ref: https://stackoverflow.com/questions/17815687/image-processing-implementing-sobel-filter
*/
func applySobelFilter(luminanceGrid [][]float64, cellWidth, cellHeight float64) [][]edgeInfo {
	rows := len(luminanceGrid)
	if rows == 0 {
		return nil
	}
	cols := len(luminanceGrid[0])
	if cols == 0 {
		return nil
	}

	edgeInfos := make([][]edgeInfo, rows)
	for y := 0; y < rows; y++ {
		edgeInfos[y] = make([]edgeInfo, cols)
	}

	sobelX := [][]int{
		{-1, 0, 1},
		{-2, 0, 2},
		{-1, 0, 1},
	}
	sobelY := [][]int{
		{-1, -2, -1},
		{0, 0, 0},
		{1, 2, 1},
	}

	//store highest value for percentile normalization
	var highestMagnitude float64 = 0
	invCellWidth := 1.0
	invCellHeight := 1.0
	if cellWidth > 0 {
		invCellWidth = 1.0 / cellWidth
	}
	if cellHeight > 0 {
		invCellHeight = 1.0 / cellHeight
	}

	for y := 1; y < rows-1; y++ {
		for x := 1; x < cols-1; x++ {

			Gx :=
				(float64(sobelX[0][0]) * luminanceGrid[y-1][x-1]) +
					(float64(sobelX[0][1]) * luminanceGrid[y-1][x]) +
					(float64(sobelX[0][2]) * luminanceGrid[y-1][x+1]) +
					(float64(sobelX[1][0]) * luminanceGrid[y][x-1]) +
					(float64(sobelX[1][1]) * luminanceGrid[y][x]) +
					(float64(sobelX[1][2]) * luminanceGrid[y][x+1]) +
					(float64(sobelX[2][0]) * luminanceGrid[y+1][x-1]) +
					(float64(sobelX[2][1]) * luminanceGrid[y+1][x]) +
					(float64(sobelX[2][2]) * luminanceGrid[y+1][x+1])

			Gy :=
				(float64(sobelY[0][0]) * luminanceGrid[y-1][x-1]) +
					(float64(sobelY[0][1]) * luminanceGrid[y-1][x]) +
					(float64(sobelY[0][2]) * luminanceGrid[y-1][x+1]) +
					(float64(sobelY[1][0]) * luminanceGrid[y][x-1]) +
					(float64(sobelY[1][1]) * luminanceGrid[y][x]) +
					(float64(sobelY[1][2]) * luminanceGrid[y][x+1]) +
					(float64(sobelY[2][0]) * luminanceGrid[y+1][x-1]) +
					(float64(sobelY[2][1]) * luminanceGrid[y+1][x]) +
					(float64(sobelY[2][2]) * luminanceGrid[y+1][x+1])

			Gx = Gx * invCellWidth
			Gy = Gy * invCellHeight
			magnitude := math.Sqrt(Gx*Gx + Gy*Gy)
			angle := math.Atan2(Gy, Gx)

			edgeInfos[y][x] = edgeInfo{
				Magnitude: magnitude,
				Angle:     angle,
			}

			if magnitude > highestMagnitude {
				highestMagnitude = magnitude
			}
		}
	}

	if highestMagnitude < 0.01 {
		highestMagnitude = 0.01
	}

	//normalize Values to 0..1
	for y := 0; y < len(edgeInfos); y++ {
		for x := 0; x < len(edgeInfos[y]); x++ {
			edgeInfos[y][x].Magnitude = edgeInfos[y][x].Magnitude / highestMagnitude
		}
	}
	_ = Logger().Info(fmt.Sprintf("Applied Sobel filter, highestMagnitude %f", highestMagnitude))

	return edgeInfos
}

// Get Rune if directionalRender is true intead of using luminance value
func getEdgeRuneFromGradient(edge edgeInfo, runeMode string) rune {
	// Sobel angle is gradient direction;
	// edge orientation is perpendicular.
	angle := edge.Angle + (math.Pi / 2)
	if angle < 0 {
		angle += 2 * math.Pi
	}

	// Normalize into 0..Pi (edges are symmetric — 0 and Pi are the same edge direction)
	if angle >= math.Pi {
		angle -= math.Pi
	}

	if runeMode != "ASCII" {
		switch {
		case angle < math.Pi/8 || angle >= 7*math.Pi/8:
			return '─'
		case angle < 3*math.Pi/8:
			return '╲'
		case angle < 5*math.Pi/8:
			return '│'
		default:
			return '╱'
		}
	}

	switch {
	case angle < math.Pi/8 || angle >= 7*math.Pi/8:
		return '-'
	case angle < 3*math.Pi/8:
		return '\\'
	case angle < 5*math.Pi/8:
		return '|'
	default:
		return '/'
	}
}

// Apply difference fo Gaussians to help with edge detections
func differenceOfGaussiansGrid(luminanceGrid [][]float64, sigma1, sigma2 float64) [][]float64 {
	rows := len(luminanceGrid)
	if rows == 0 {
		return nil
	}
	cols := len(luminanceGrid[0])
	if cols == 0 {
		return nil
	}

	if sigma1 <= 0 {
		sigma1 = 0.6
	}
	if sigma2 <= sigma1 {
		sigma2 = sigma1 * 2
	}

	clampInt := func(x, lo, hi int) int {
		if x < lo {
			return lo
		}
		if x > hi {
			return hi
		}
		return x
	}

	gaussianKernel1D := func(sigma float64) ([]float64, int) {
		if sigma <= 0 {
			return []float64{1}, 0
		}
		radius := int(math.Ceil(3 * sigma))
		size := 2*radius + 1

		k := make([]float64, size)
		var sum float64
		twoSigma2 := 2 * sigma * sigma

		for i := -radius; i <= radius; i++ {
			x := float64(i)
			v := math.Exp(-(x * x) / twoSigma2)
			k[i+radius] = v
			sum += v
		}

		if sum < 1e-12 {
			sum = 1e-12
		}
		for i := range k {
			k[i] /= sum
		}

		return k, radius
	}

	gaussianBlur := func(grid [][]float64, sigma float64) [][]float64 {
		k, r := gaussianKernel1D(sigma)

		// horizontal pass
		tmp := make([][]float64, rows)
		for y := 0; y < rows; y++ {
			tmp[y] = make([]float64, cols)
			for x := 0; x < cols; x++ {
				sum := 0.0
				for i := -r; i <= r; i++ {
					xx := clampInt(x+i, 0, cols-1)
					sum += grid[y][xx] * k[i+r]
				}
				tmp[y][x] = sum
			}
		}

		// vertical pass
		out := make([][]float64, rows)
		for y := 0; y < rows; y++ {
			out[y] = make([]float64, cols)
			for x := 0; x < cols; x++ {
				sum := 0.0
				for i := -r; i <= r; i++ {
					yy := clampInt(y+i, 0, rows-1)
					sum += tmp[yy][x] * k[i+r]
				}
				out[y][x] = sum
			}
		}

		return out
	}

	// compute DoG = blur(sigma1) - blur(sigma2)
	g1 := gaussianBlur(luminanceGrid, sigma1)
	g2 := gaussianBlur(luminanceGrid, sigma2)

	dog := make([][]float64, rows)
	var maxAbs float64

	for y := 0; y < rows; y++ {
		dog[y] = make([]float64, cols)
		for x := 0; x < cols; x++ {
			v := g1[y][x] - g2[y][x]
			dog[y][x] = v
			av := math.Abs(v)
			if av > maxAbs {
				maxAbs = av
			}
		}
	}
	return dog
}
