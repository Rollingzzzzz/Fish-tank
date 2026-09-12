// F18/F19: darkscan — the black-blob detector. Scans a PNG for large
// connected near-black regions and reports their bounding boxes and fill
// ratios, so "weird blackness" artifacts are found by measurement instead
// of eyeballing. Report tool by default; -strict exits 1 when a region
// exceeds the size thresholds.
package main

import (
	"fmt"
	"image"
	"image/png"
	"os"
	"sort"
)

type region struct {
	x0, y0, x1, y1 int
	area           int
}

func (r region) w() int { return r.x1 - r.x0 + 1 }
func (r region) h() int { return r.y1 - r.y0 + 1 }

func main() {
	thresh, minW, minH := 14.0, 300, 80
	strict := false
	var files []string
	for _, a := range os.Args[1:] {
		switch a {
		case "-strict":
			strict = true
		default:
			files = append(files, a)
		}
	}
	if len(files) == 0 {
		fmt.Println("usage: darkscan [-strict] <png...>   (thresholds: lum<14, region>=300x80)")
		os.Exit(2)
	}
	bad := 0
	for _, f := range files {
		bad += scan(f, thresh, minW, minH)
	}
	if strict && bad > 0 {
		os.Exit(1)
	}
}

func scan(path string, thresh float64, minW, minH int) int {
	f, err := os.Open(path)
	if err != nil {
		fmt.Println("darkscan:", err)
		return 1
	}
	defer f.Close()
	p, err := png.Decode(f)
	if err != nil {
		fmt.Println("darkscan:", path, err)
		return 1
	}
	b := p.Bounds()
	seen := make([]bool, b.Dx()*b.Dy())
	dark := func(x, y int) bool {
		r, g, bl, _ := p.At(x, y).RGBA()
		return float64(r>>8)*0.299+float64(g>>8)*0.587+float64(bl>>8)*0.114 < thresh
	}
	var regs []region
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			i := (y-b.Min.Y)*b.Dx() + (x - b.Min.X)
			if seen[i] || !dark(x, y) {
				continue
			}
			// BFS flood fill
			reg := region{x0: x, y0: y, x1: x, y1: y}
			queue := []image.Point{{x, y}}
			seen[i] = true
			for len(queue) > 0 {
				q := queue[len(queue)-1]
				queue = queue[:len(queue)-1]
				reg.area++
				if q.X < reg.x0 {
					reg.x0 = q.X
				}
				if q.X > reg.x1 {
					reg.x1 = q.X
				}
				if q.Y < reg.y0 {
					reg.y0 = q.Y
				}
				if q.Y > reg.y1 {
					reg.y1 = q.Y
				}
				for _, d := range [4]image.Point{{1, 0}, {-1, 0}, {0, 1}, {0, -1}} {
					nx, ny := q.X+d.X, q.Y+d.Y
					if nx < b.Min.X || ny < b.Min.Y || nx >= b.Max.X || ny >= b.Max.Y {
						continue
					}
					ni := (ny-b.Min.Y)*b.Dx() + (nx - b.Min.X)
					if !seen[ni] && dark(nx, ny) {
						seen[ni] = true
						queue = append(queue, image.Point{nx, ny})
					}
				}
			}
			regs = append(regs, reg)
		}
	}
	sort.Slice(regs, func(i, j int) bool { return regs[i].area > regs[j].area })
	fmt.Printf("darkscan %s: %d dark regions (lum<%.0f)\n", path, len(regs), thresh)
	bad := 0
	for i, r := range regs {
		if i >= 10 {
			break
		}
		fill := float64(r.area) / float64(r.w()*r.h())
		flag := ""
		if r.w() >= minW && r.h() >= minH {
			flag = "  <-- LARGE"
			bad++
		}
		fmt.Printf("  %dx%d at +%d+%d  fill %.2f  area %d%s\n", r.w(), r.h(), r.x0, r.y0, fill, r.area, flag)
	}
	return bad
}
