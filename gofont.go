package gofont

import (
	"github.com/gonutz/fontstash.go/truetype"
	"image"
	"image/color"
	"image/draw"
	"io/ioutil"
)

func LoadFromFile(path string) (*Font, error) {
	data, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}

	info, err := truetype.InitFont(data, truetype.GetFontOffsetForIndex(data, 0))
	if err != nil {
		return nil, err
	}

	return &Font{
		info,
		make(map[int]map[rune]*image.Alpha),
		0, 0, 0,
		20,
	}, nil
}

type Font struct {
	fontInfo    *truetype.FontInfo
	letters     map[int]map[rune]*image.Alpha
	R, G, B     uint8
	PixelHeight int
}

func (f *Font) Write(text string, dest draw.Image, startX, startY int) (newX, newY int) {
	scale := f.fontInfo.ScaleForPixelHeight(float64(f.PixelHeight))
	ascend, descend, baseline := f.fontInfo.GetFontVMetrics()

	source := image.NewUniform(color.RGBA{f.R, f.G, f.B, 255})
	x := startX
	yOffset := round(float64(ascend+baseline) * scale)
	y := startY + yOffset

	var last rune
	for i, r := range text {
		if r == '\n' {
			x = startX
			y += round(float64(ascend-descend+baseline) * scale)
			continue
		}

		letter := f.getLetter(r, scale)
		advance, leftSideBearing := f.fontInfo.GetCodepointHMetrics(int(r))
		x0, _, _, y1 := f.fontInfo.GetCodepointBitmapBox(int(r), 0, scale)
		x += round(float64(leftSideBearing) * scale)
		kerning := 0
		if i != 0 {
			kerning = round(float64(f.fontInfo.GetCodepointKernAdvance(int(last), int(r))) * scale)
		}
		draw.DrawMask(dest,
			image.Rect(x+x0, y+y1-letter.Bounds().Dy(), x+letter.Bounds().Dx(), y+letter.Bounds().Dy()),
			source, image.ZP, letter, image.ZP, draw.Over)
		x += round(float64(advance)*scale) + kerning
	}
	return x, y - yOffset
}

func (f *Font) getLetter(r rune, scale float64) *image.Alpha {
	size, ok := f.letters[f.PixelHeight]
	if !ok {
		size = make(map[rune]*image.Alpha)
		f.letters[f.PixelHeight] = size
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
