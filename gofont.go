package gofont

import (
	"image"
	"image/color"
	"image/draw"
	"io"
	"io/ioutil"
	"os"
	"strings"

	"github.com/gonutz/fontstash.go/truetype"
)

// Read reads the True Type Font (.ttf) and creates a font from it. The returned
// font is solid black and 20 pixels high.
func Read(r io.Reader) (*Font, error) {
	data, err := ioutil.ReadAll(r)
	if err != nil {
		return nil, err
	}

	info, err := truetype.InitFont(data, truetype.GetFontOffsetForIndex(data, 0))
	if err != nil {
		return nil, err
	}

	return &Font{
		A:              255,
		HeightInPixels: 20,
		fontInfo:       info,
		letters:        make(map[int]map[rune]*image.Alpha),
	}, nil
}

// LoadFromFile loads a True Type Font file (.ttf) and creates a font from it.
// The returned font is solid black and 20 pixels high.
func LoadFromFile(path string) (*Font, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	return Read(f)
}

// Font contains a font face, color and size information. To change the font
// color, set the R, G, B, A values. These are red/green/blue/opacity color
// channels in the range [0..255]. A=0 is fully transparent, A=255 is solid. To
// change the font size, set the HeightInPixels.
type Font struct {
	R, G, B, A     uint8
	HeightInPixels int

	fontInfo *truetype.FontInfo
	// letters maps from a pixel size to a map from characters to their glyph
	// images: letters[sizeInPixels][character] == alphaImage
	letters map[int]map[rune]*image.Alpha
}

// Measure returns the size of the text when written in the Font's current pixel
// height.
func (f *Font) Measure(text string) (w, h int) {
	scale := f.fontInfo.ScaleForPixelHeight(float64(f.HeightInPixels))
	ascend, descend, baseline := f.fontInfo.GetFontVMetrics()
	lineHeight := round(float64(ascend-descend+baseline) * scale)

	x := 0
	yOffset := round(float64(ascend+baseline) * scale)
	y := 0 + yOffset

	var last rune
	for i, r := range text {
		if r == '\n' {
			x = 0
			y += lineHeight
			continue
		}

		advance, leftSideBearing := f.fontInfo.GetCodepointHMetrics(int(r))
		x += round(float64(leftSideBearing) * scale)
		kerning := 0
		if i != 0 {
			kerning = round(float64(f.fontInfo.GetCodepointKernAdvance(int(last), int(r))) * scale)
		}
		x += round(float64(advance)*scale) + kerning
		last = r
	}
	return x, y - yOffset + lineHeight
}

// WriteAnchor lets you justify the text horizontally and vertically. The anchor
// determines the point that the text "gravitates to".
// Here are some examples of what the Anchor type means, 'X' marks the anchor
// point in the image:
//
//     AnchorCenter    AnchorCenterLeft  AnchorBottomRight
//     +----------+      +----------+      +----------+
//     |          |      |          |      |          |
//     |   this   |      |centered  |      |          |
//     |    iX    |      Xleft      |      |   text at|
//     | centered |      |vertically|      |    bottom|
//     |          |      |          |      |     right|
//     +----------+      +----------+      +----------X
func (f *Font) WriteAnchor(dest draw.Image, text string, anchorX, anchorY int, anchor Anchor) {
	lines := strings.Split(text, "\n")
	scale := f.fontInfo.ScaleForPixelHeight(float64(f.HeightInPixels))
	ascend, descend, baseline := f.fontInfo.GetFontVMetrics()
	yOffset := round(float64(ascend+baseline) * scale)
	lineHeight := round(float64(ascend-descend+baseline) * scale)
	source := image.NewUniform(color.RGBA{f.R, f.G, f.B, 255})

	y := anchorY
	if anchor == AnchorCenterLeft ||
		anchor == AnchorCenter ||
		anchor == AnchorCenterRight {
		// vertically centered
		y = anchorY - len(lines)*lineHeight/2
	}
	if anchor == AnchorBottomLeft ||
		anchor == AnchorBottomCenter ||
		anchor == AnchorBottomRight {
		// anchored at the bottom
		y = anchorY - len(lines)*lineHeight
	}
	y += yOffset

	for _, line := range lines {
		x := anchorX
		if anchor == AnchorTopCenter ||
			anchor == AnchorCenter ||
			anchor == AnchorBottomCenter {
			// horizontally centered
			w, _ := f.Measure(line)
			x = anchorX - w/2
		}
		if anchor == AnchorTopRight ||
			anchor == AnchorCenterRight ||
			anchor == AnchorBottomRight {
			// anchored to the right
			w, _ := f.Measure(line)
			x = anchorX - w
		}

		var last rune
		for i, r := range line {
			advance, leftSideBearing := f.fontInfo.GetCodepointHMetrics(int(r))
			x += round(float64(leftSideBearing) * scale)

			letter := f.getLetter(r, scale)
			var mask image.Image = letter
			if f.A != 255 {
				mask = newAlphaMultiplied(letter, f.A)
			}
			w := letter.Bounds().Dx()
			h := letter.Bounds().Dy()
			x0, _, _, y1 := f.fontInfo.GetCodepointBitmapBox(int(r), 0, scale)
			draw.DrawMask(
				dest, image.Rect(x+x0, y+y1-h, x+w, y+h),
				source, image.ZP,
				mask, image.ZP,
				draw.Over,
			)
			kerning := 0
			if i != 0 {
				kerning = round(float64(f.fontInfo.GetCodepointKernAdvance(int(last), int(r))) * scale)
			}
			x += round(float64(advance)*scale) + kerning

			last = r
		}

		y += lineHeight
	}
}

type Anchor int

const (
	AnchorTopLeft Anchor = iota
	AnchorCenterLeft
	AnchorBottomLeft
	AnchorTopCenter
	AnchorCenter
	AnchorBottomCenter
	AnchorTopRight
	AnchorCenterRight
	AnchorBottomRight
)

// Write writes the given text aligned top-left at the given image position. It
// returns the position where the text ends. This can be used to write the next
// text, e.g. if you want to write a single word in a text with a different
// color.
func (f *Font) Write(dest draw.Image, text string, startX, startY int) (newX, newY int) {
	scale := f.fontInfo.ScaleForPixelHeight(float64(f.HeightInPixels))
	ascend, descend, baseline := f.fontInfo.GetFontVMetrics()
	lineHeight := round(float64(ascend-descend+baseline) * scale)

	source := image.NewUniform(color.RGBA{f.R, f.G, f.B, 255})
	x := startX
	yOffset := round(float64(ascend+baseline) * scale)
	y := startY + yOffset

	var last rune
	for i, r := range text {
		if r == '\n' {
			x = startX
			y += lineHeight
			last = 0
			continue
		}

		advance, leftSideBearing := f.fontInfo.GetCodepointHMetrics(int(r))
		x += round(float64(leftSideBearing) * scale)

		letter := f.getLetter(r, scale)
		var mask image.Image = letter
		if f.A != 255 {
			mask = newAlphaMultiplied(letter, f.A)
		}
		w := letter.Bounds().Dx()
		h := letter.Bounds().Dy()
		x0, _, _, y1 := f.fontInfo.GetCodepointBitmapBox(int(r), 0, scale)
		draw.DrawMask(
			dest, image.Rect(x+x0, y+y1-h, x+w, y+h),
			source, image.ZP,
			mask, image.ZP,
			draw.Over,
		)
		kerning := 0
		if i != 0 {
			kerning = round(float64(f.fontInfo.GetCodepointKernAdvance(int(last), int(r))) * scale)
		}
		x += round(float64(advance)*scale) + kerning
		last = r
	}
	return x, y - yOffset
}

func (f *Font) getLetter(r rune, scale float64) *image.Alpha {
	size, ok := f.letters[f.HeightInPixels]
	if !ok {
		size = make(map[rune]*image.Alpha)
		f.letters[f.HeightInPixels] = size
	}

	letter, ok := size[r]
	if !ok {
		pixels, w, h := f.fontInfo.GetCodepointBitmap(0, scale, int(r), 0, 0)
		letter = image.NewAlpha(image.Rect(0, 0, w, h))
		for y := 0; y < h; y++ {
			copy(letter.Pix[letter.PixOffset(0, y):], pixels[y*w:y*w+w])
		}
	}

	return letter
}

func round(f float64) int {
	if f < 0 {
		return int(f - 0.5)
	}
	return int(f + 0.5)
}

func newAlphaMultiplied(img *image.Alpha, a uint8) image.Image {
	return alphaMultiplied{
		Alpha: img,
		a:     int(a),
	}
}

type alphaMultiplied struct {
	*image.Alpha
	a int
}

func (img alphaMultiplied) At(x, y int) color.Color {
	orig := img.AlphaAt(x, y).A
	return color.Alpha{A: uint8(int(orig) * img.a / 255)}
}
