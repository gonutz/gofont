# gofont

`gofont` is a Go library to write text on Go's standard `draw.Image`. It loads True Type Fonts (TTF) and lets you pick a (potentially transparent) color and size to write text in a formatted way.

Here is a link to the [Godoc reference](https://godoc.org/github.com/gonutz/gofont).

# Example

This example image is created by the following example code:

![Example](https://github.com/gonutz/gofont/blob/master/example/example.png)

```Go
package main

import (
	"bytes"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"os"

	"golang.org/x/image/font/gofont/goregular"

	"github.com/gonutz/gofont"
)

func main() {
	// The font can be read from an io.Reader or loaded from file.
	//  On Windows you could also do:
	// font, err := gofont.LoadFromFile("c:/windows/fonts/arial.ttf")
	font, err := gofont.Read(bytes.NewReader(goregular.TTF))
	check(err)

	// Create a solid black image.
	img := image.NewRGBA(image.Rect(0, 0, 300, 200))
	clearToBlack(img)

	// Write some text, by default Write will align it top-left.
	backgroundText := `This is some text with line
breaks in it. \n is used for
line breaks; do not place
any \r in the string, even
if you are on Windows ;-)`
	font.HeightInPixels = 25
	font.R, font.G, font.B, font.A = 0, 255, 0, 255 // solid green
	font.Write(img, backgroundText, 0, 0)

	// Place a larger, semi-transparent text, centered vertically and
	// horizontally.
	overlayText := `Centered
overlay`
	font.HeightInPixels = 50
	font.R, font.G, font.B, font.A = 255, 255, 0, 128 // half-transparent yellow
	font.WriteAnchor(img, overlayText, 150, 100, gofont.AnchorCenter)

	// Write the image to a file.
	f, err := os.Create("example.png")
	check(err)
	defer f.Close()
	check(png.Encode(f, img))
}

func clearToBlack(img draw.Image) {
	draw.Draw(img, img.Bounds(), image.NewUniform(color.Black), image.ZP, draw.Src)
}

func check(err error) {
	if err != nil {
		panic(err)
	}
}
```