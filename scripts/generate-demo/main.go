package main

import (
	"image"
	"image/color"
	"image/draw"
	"image/gif"
	"os"
	"path/filepath"

	"golang.org/x/image/font"
	"golang.org/x/image/font/basicfont"
	"golang.org/x/image/math/fixed"
)

var palette = color.Palette{
	color.RGBA{R: 15, G: 17, B: 22, A: 255},
	color.RGBA{R: 204, G: 211, B: 222, A: 255},
	color.RGBA{R: 111, G: 209, B: 156, A: 255},
	color.RGBA{R: 102, G: 112, B: 133, A: 255},
	color.RGBA{R: 245, G: 189, B: 97, A: 255},
}

func main() {
	output := "assets/demo.gif"
	if len(os.Args) > 1 {
		output = os.Args[1]
	}
	if err := os.MkdirAll(filepath.Dir(output), 0755); err != nil {
		panic(err)
	}

	states := []struct {
		title string
		log   []string
	}{
		{"Panestra CLI - Waiting for prompt", []string{"$ codex", "", "  Codex is ready. Type a prompt to begin."}},
		{"Task: Refactor the authentication flow", []string{"$ codex", "> Refactor the authentication flow", "", "  Inspecting the current implementation...", "  Updating auth/session.go", "  Running tests..."}},
		{"Task: Update the existing tests too", []string{"$ codex", "> Refactor the authentication flow", "", "  Authentication flow updated.", "> Update the existing tests too", "", "  Updating auth/session_test.go", "  All tests passed."}},
	}

	animation := &gif.GIF{LoopCount: 0}
	for i, state := range states {
		for frame := 0; frame < 8; frame++ {
			animation.Image = append(animation.Image, render(state.title, state.log, frame%2 == 0))
			delay := 10
			if frame == 7 {
				delay = []int{80, 110, 140}[i]
			}
			animation.Delay = append(animation.Delay, delay)
		}
	}

	f, err := os.Create(output)
	if err != nil {
		panic(err)
	}
	defer f.Close()
	if err := gif.EncodeAll(f, animation); err != nil {
		panic(err)
	}
}

func render(title string, lines []string, cursor bool) *image.Paletted {
	const smallWidth, smallHeight, scale = 480, 240, 2
	small := image.NewPaletted(image.Rect(0, 0, smallWidth, smallHeight), palette)
	draw.Draw(small, small.Bounds(), &image.Uniform{C: palette[0]}, image.Point{}, draw.Src)

	line(small, 0, 0, smallWidth, 0, 2)
	text(small, 8, 13, title, 2)
	text(small, 15, 40, "PANESTRA CLI  /  CODEX", 3)
	for i, value := range lines {
		colour := uint8(1)
		if len(value) > 0 && value[0] == '>' {
			colour = 4
		}
		text(small, 15, 70+i*18, value, colour)
	}
	if cursor {
		y := 76 + len(lines)*18
		draw.Draw(small, image.Rect(15, y, 21, y+2), &image.Uniform{C: palette[2]}, image.Point{}, draw.Src)
	}
	text(small, 15, 225, "Latest prompt stays visible while output scrolls.", 3)

	large := image.NewPaletted(image.Rect(0, 0, smallWidth*scale, smallHeight*scale), palette)
	for y := 0; y < smallHeight; y++ {
		for x := 0; x < smallWidth; x++ {
			index := small.ColorIndexAt(x, y)
			for dy := 0; dy < scale; dy++ {
				for dx := 0; dx < scale; dx++ {
					large.SetColorIndex(x*scale+dx, y*scale+dy, index)
				}
			}
		}
	}
	return large
}

func text(dst draw.Image, x, y int, value string, colour uint8) {
	d := font.Drawer{
		Dst:  dst,
		Src:  &image.Uniform{C: palette[colour]},
		Face: basicfont.Face7x13,
		Dot:  fixed.P(x, y),
	}
	d.DrawString(value)
}

func line(dst draw.Image, x1, y1, x2, y2 int, colour uint8) {
	draw.Draw(dst, image.Rect(x1, y1, x2, y2+1), &image.Uniform{C: palette[colour]}, image.Point{}, draw.Src)
}
