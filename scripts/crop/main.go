// crop — zoom helper for PNG self-review.
package main

import (
	"fmt"
	"image"
	"image/png"
	"os"
	"strconv"
)

func main() {
	in, out := os.Args[1], os.Args[2]
	x0, _ := strconv.Atoi(os.Args[3])
	y0, _ := strconv.Atoi(os.Args[4])
	x1, _ := strconv.Atoi(os.Args[5])
	y1, _ := strconv.Atoi(os.Args[6])
	f, err := os.Open(in)
	if err != nil {
		panic(err)
	}
	img, _, err := image.Decode(f)
	if err != nil {
		panic(err)
	}
	f.Close()
	sub := img.(interface {
		SubImage(r image.Rectangle) image.Image
	}).SubImage(image.Rect(x0, y0, x1, y1))
	w, err := os.Create(out)
	if err != nil {
		panic(err)
	}
	defer w.Close()
	if err := png.Encode(w, sub); err != nil {
		panic(err)
	}
	fmt.Println("wrote", out)
}
