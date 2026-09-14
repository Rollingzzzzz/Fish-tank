// G93 eyepair — the eye-pair metric for glass-gaze evidence. Scans a PNG
// for the two dark eye clusters of a face-on stare: dark pixels cluster,
// the two biggest clusters must sit on opposite sides of the fish's body
// axis at matching heights (symmetry). Exit 0 = a face-on stare detected.
//
// Usage: go run ./scripts/eyepair shot.png [more.png ...]
package main

import (
	"fmt"
	"image"
	"image/png"
	"os"
	"sort"
)

type blob struct {
	n      int
	cx, cy float64
}

func main() {
	if len(os.Args) < 2 {
		fmt.Println("usage: eyepair <png> [png...]")
		os.Exit(2)
	}
	ok := true
	for _, path := range os.Args[1:] {
		if !scan(path) {
			ok = false
		}
	}
	if !ok {
		os.Exit(1)
	}
}

func scan(path string) bool {
	f, err := os.Open(path)
	if err != nil {
		fmt.Println("open:", err)
		return false
	}
	defer f.Close()
	img, err := png.Decode(f)
	if err != nil {
		fmt.Println("decode:", err)
		return false
	}
	b := img.Bounds()
	w, h := b.Max.X-b.Min.X, b.Max.Y-b.Min.Y
	const cell = 2 // 2px grid cells keep the union-find small
	gw, gh := w/cell, h/cell
	dark := make([]bool, gw*gh)
	for gy := 0; gy < gh; gy++ {
		for gx := 0; gx < gw; gx++ {
			px := img.At(b.Min.X+gx*cell, b.Min.Y+gy*cell)
			r, g, bl, _ := px.RGBA()
			if uint8(r>>8)+uint8(g>>8)+uint8(bl>>8) < 90 {
				dark[gy*gw+gx] = true
			}
		}
	}
	// connected components (4-neighbourhood, iterative flood)
	label := make([]int, gw*gh)
	var blobs []blob
	for i := range dark {
		if !dark[i] || label[i] != 0 {
			continue
		}
		id := len(blobs) + 1
		stack := []int{i}
		label[i] = id
		var n int
		var sx, sy float64
		for len(stack) > 0 {
			j := stack[len(stack)-1]
			stack = stack[:len(stack)-1]
			jx, jy := j%gw, j/gw
			n++
			sx += float64(jx)
			sy += float64(jy)
			for _, d := range []int{-1, 1, -gw, gw} {
				k := j + d
				if k < 0 || k >= len(dark) || !dark[k] || label[k] != 0 {
					continue
				}
				if d == -1 && j%gw == 0 || d == 1 && j%gw == gw-1 {
					continue
				}
				label[k] = id
				stack = append(stack, k)
			}
		}
		blobs = append(blobs, blob{n: n, cx: sx / float64(n), cy: sy / float64(n)})
	}
	sort.Slice(blobs, func(i, j int) bool { return blobs[i].n > blobs[j].n })
	if len(blobs) < 2 {
		fmt.Printf("%s: NO eye pair (%d dark clusters)\n", path, len(blobs))
		return false
	}
	a, b2 := blobs[0], blobs[1]
	ax, ay := a.cx*cell, a.cy*cell
	bx, by := b2.cx*cell, b2.cy*cell
	sep := ax - bx
	if sep < 0 {
		sep = -sep
	}
	sym := ay - by
	if sym < 0 {
		sym = -sym
	}
	// the pair must be well separated horizontally and level vertically —
	// two eyes, not one eye and a fin tip
	pair := sep > 8 && sym < 12 && a.n >= 3 && b2.n >= 3
	fmt.Printf("%s: clusters=%d eye1=(%.0f,%.0f,%dpx) eye2=(%.0f,%.0f,%dpx) sep=%.0f sym=%.0f → %s\n",
		path, len(blobs), ax, ay, a.n, bx, by, b2.n, sep, sym, verdict(pair))
	_ = image.Point{}
	return pair
}

func verdict(ok bool) string {
	if ok {
		return "FACE-ON STARE"
	}
	return "NOT A PAIR"
}
