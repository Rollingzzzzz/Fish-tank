// pngstats — F2 acceptance helper: prints the share of pixels above a
// luminance threshold plus basic stats for a PNG frame.
// Usage: go run ./scripts/pngstats <image.png> [threshold]
package main

import (
	"fmt"
	"image"
	_ "image/png"
	"os"
	"strconv"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("usage: pngstats <image.png> [threshold]")
		os.Exit(2)
	}
	f, err := os.Open(os.Args[1])
	if err != nil {
		panic(err)
	}
	defer f.Close()
	img, _, err := image.Decode(f)
	if err != nil {
		panic(err)
	}
	th := 240.0
	if len(os.Args) > 2 {
		th, _ = strconv.ParseFloat(os.Args[2], 64)
	}
	b := img.Bounds()
	total, hot, sum := 0, 0, 0.0
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			r, g, bl, _ := img.At(x, y).RGBA()
			lum := 0.2126*float64(r>>8) + 0.7152*float64(g>>8) + 0.0722*float64(bl>>8)
			sum += lum
			total++
			if lum > th {
				hot++
			}
		}
	}
	fmt.Printf("%s: size=%dx%d meanLum=%.1f hot(>%.0f)=%.3f%%\n",
		os.Args[1], b.Dx(), b.Dy(), sum/float64(total), th, 100*float64(hot)/float64(total))
}
