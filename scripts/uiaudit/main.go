// F19: uiaudit — the visibility gate. Audits -shot evidence PNGs against
// their manifest of expected control rects: every control must sit fully
// inside the frame, contain pixels that differ from its surroundings, and
// contain bright (label/accent) pixels. Exit 1 on any failure, so it can
// gate scripts/check and CI like any other test.
package main

import (
	"encoding/json"
	"fmt"
	"image"
	"image/png"
	"os"
	"path/filepath"
	"sort"
)

type rect struct {
	Name  string `json:"name"`
	X     int    `json:"x"`
	Y     int    `json:"y"`
	W     int    `json:"w"`
	H     int    `json:"h"`
	Image string `json:"image"`
	Kind  string `json:"kind,omitempty"` // "" / "control" = strict, "scene" = diff-only
}

func lum(c interface{ RGBA() (r, g, b, a uint32) }) float64 {
	r, g, b, _ := c.RGBA()
	return float64(r>>8)*0.299 + float64(g>>8)*0.587 + float64(b>>8)*0.114
}

func main() {
	if len(os.Args) < 2 {
		fmt.Println("usage: uiaudit <dir-with-manifest.json> [more dirs...]")
		os.Exit(2)
	}
	minDiff, minBright := 0.02, 0.005
	failed := 0
	checked := 0
	for _, dir := range os.Args[1:] {
		failed += auditDir(dir, minDiff, minBright, &checked)
	}
	fmt.Printf("uiaudit: %d rects checked, %d failed\n", checked, failed)
	if failed > 0 {
		os.Exit(1)
	}
}

func auditDir(dir string, minDiff, minBright float64, checked *int) int {
	mb, err := os.ReadFile(filepath.Join(dir, "manifest.json"))
	if err != nil {
		fmt.Println("uiaudit:", err)
		return 1
	}
	var rects []rect
	if err := json.Unmarshal(mb, &rects); err != nil {
		fmt.Println("uiaudit: bad manifest:", err)
		return 1
	}
	byImage := map[string][]rect{}
	var order []string
	for _, r := range rects {
		if _, seen := byImage[r.Image]; !seen {
			order = append(order, r.Image)
		}
		byImage[r.Image] = append(byImage[r.Image], r)
	}
	sort.Strings(order)
	failed := 0
	for _, img := range order {
		m := image.NewRGBA(image.Rect(0, 0, 0, 0))
		f, err := os.Open(filepath.Join(dir, img))
		if err != nil {
			fmt.Printf("FAIL %s %s: %v\n", img, "(open)", err)
			failed++
			continue
		}
		p, err := png.Decode(f)
		f.Close()
		if err != nil {
			fmt.Printf("FAIL %s %s: %v\n", img, "(decode)", err)
			failed++
			continue
		}
		m = toRGBA(p)
		for _, r := range byImage[img] {
			*checked++
			if !auditRect(m, r, minDiff, minBright) {
				failed++
			}
		}
	}
	return failed
}

func toRGBA(p image.Image) *image.RGBA {
	b := p.Bounds()
	m := image.NewRGBA(image.Rect(0, 0, b.Dx(), b.Dy()))
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			m.Set(x-b.Min.X, y-b.Min.Y, p.At(x, y))
		}
	}
	return m
}

// auditRect checks one control rect; prints PASS/FAIL with numbers.
func auditRect(m *image.RGBA, r rect, minDiff, minBright float64) bool {
	b := m.Bounds()
	if r.X < 0 || r.Y < 0 || r.X+r.W > b.Dx() || r.Y+r.H > b.Dy() {
		fmt.Printf("FAIL %-28s (%s): out of frame %dx%d+%d+%d on %dx%d\n",
			r.Name, r.Image, r.W, r.H, r.X, r.Y, b.Dx(), b.Dy())
		return false
	}
	if r.W < 2 || r.H < 2 {
		fmt.Printf("FAIL %-28s (%s): degenerate rect\n", r.Name, r.Image)
		return false
	}
	// background estimate: median luminance of a ring around the rect
	var ring []float64
	for x := r.X - 3; x < r.X+r.W+3; x++ {
		for y := r.Y - 3; y < r.Y+r.H+3; y++ {
			if x >= 0 && y >= 0 && x < b.Dx() && y < b.Dy() &&
				(x < r.X || x >= r.X+r.W || y < r.Y || y >= r.Y+r.H) {
				ring = append(ring, lum(m.At(x, y)))
			}
		}
	}
	bg := median(ring)
	n, diff, bright := 0, 0, 0
	for y := r.Y; y < r.Y+r.H; y++ {
		for x := r.X; x < r.X+r.W; x++ {
			l := lum(m.At(x, y))
			n++
			if absF(l-bg) >= 30 {
				diff++
			}
			if l >= 120 { // readable label on ~15-25 lum panels = 6x contrast
				bright++
			}
		}
	}
	dp, bp := float64(diff)/float64(n), float64(bright)/float64(n)
	// scene rects only need meaningful content; controls must also carry
	// bright (readable) label/accent pixels.
	ok := dp >= 0.03
	if r.Kind != "scene" {
		ok = dp >= minDiff && bp >= minBright
	}
	verdict := "PASS"
	if !ok {
		verdict = "FAIL"
	}
	fmt.Printf("%s %-28s (%s): diff %.1f%% bright %.1f%% vs bg %.0f\n",
		verdict, r.Name, r.Image, dp*100, bp*100, bg)
	return ok
}

func median(v []float64) float64 {
	if len(v) == 0 {
		return 0
	}
	s := append([]float64{}, v...)
	sort.Float64s(s)
	return s[len(s)/2]
}

func absF(v float64) float64 {
	if v < 0 {
		return -v
	}
	return v
}
