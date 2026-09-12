// G3.6: C6 canonical fingerprints and the uniqueness acceptance gate.
// Fingerprint = contract.Fingerprint(kind, canonical) where canonical covers
// pattern type, density/size quantized to 0.05 buckets and palette hues
// quantized to 10-degree buckets (hue extraction implemented here, stdlib only).
package content

import (
	"fmt"
	"math"
	"strings"

	"github.com/Rollingzzzzz/Fish-tank/internal/contract"
)

// rgbHue converts 0..1 RGB components to a hue in [0,360). Grays yield 0.
func rgbHue(r, g, b float64) float64 {
	max := math.Max(r, math.Max(g, b))
	min := math.Min(r, math.Min(g, b))
	if max == min {
		return 0
	}
	d := max - min
	var h float64
	switch max {
	case r:
		h = math.Mod((g-b)/d, 6)
	case g:
		h = (b-r)/d + 2
	default:
		h = (r-g)/d + 4
	}
	h *= 60
	if h < 0 {
		h += 360
	}
	return h
}

// hexHue extracts the hue bucket (0..35, 10-degree buckets) of a #rrggbb color.
func hexHue(hex string) int {
	return int(math.Round(hexHueDeg(hex)/contract.HueQuantum)) % int(360/contract.HueQuantum)
}

// hexHueDeg returns the exact hue of a #rrggbb color in [0,360); grays yield 0.
func hexHueDeg(hex string) float64 {
	r, g, b := contract.HexToRGB(hex)
	return rgbHue(float64(r)/255, float64(g)/255, float64(b)/255)
}

// quantF snaps a float to the 0.05 fingerprint bucket as a stable string.
func quantF(v float64) string {
	return fmt.Sprintf("%.2f", contract.Quantize(v, contract.FingerprintQuantum))
}

func joinPipe(parts ...string) string { return strings.Join(parts, "|") }

// hueCanonical is the palette-hue-only canonical shared by water and plant
// kinds (their gate inputs carry no pattern data).
func hueCanonical(prefix string, pal contract.Palette) string {
	return joinPipe(prefix,
		itoa(hexHue(pal.Body)), itoa(hexHue(pal.Belly)),
		itoa(hexHue(pal.Accent)), itoa(hexHue(pal.Glow)))
}

func speciesCanonical(pat contract.Pattern, pal contract.Palette) string {
	return joinPipe(pat.Type, quantF(pat.Density), quantF(pat.Size),
		itoa(hexHue(pal.Body)), itoa(hexHue(pal.Belly)),
		itoa(hexHue(pal.Accent)), itoa(hexHue(pal.Glow)))
}

func waterCanonical(w *contract.WaterPreset) string {
	return hueCanonical("water", WaterPalette(w))
}

func plantCanonical(p *contract.PlantDesign) string {
	return hueCanonical("plant", PlantPalette(p))
}

// coralCanonical covers kind, frond count, quantized curve/sway and the four
// gradient hue buckets — the write path and GateCoral must stay identical.
func coralCanonical(c *contract.CoralDesign) string {
	pal := CoralPalette(c)
	return joinPipe("coral", c.Kind, itoa(c.Fronds), quantF(c.Curve), quantF(c.Sway),
		itoa(hexHue(pal.Body)), itoa(hexHue(pal.Belly)),
		itoa(hexHue(pal.Accent)), itoa(hexHue(pal.Glow)))
}

// recipeCanonical covers stage and the stage multipliers (quantized) as well
// as the pattern, because recipes have no palette of their own.
func recipeCanonical(r *contract.PatternRecipe) string {
	return joinPipe("pattern", r.Stage, quantF(r.SatMul), quantF(r.AlphaMul), quantF(r.GlowMul),
		r.Pattern.Type, quantF(r.Pattern.Density), quantF(r.Pattern.Size))
}

// WaterPalette maps a water preset's colors into a Palette for the gate.
func WaterPalette(w *contract.WaterPreset) contract.Palette {
	return contract.Palette{Body: w.TopColor, Belly: w.BottomColor, Accent: w.Accent, Glow: w.Accent}
}

// PlantPalette maps a plant's gradient colors into a Palette for the gate.
func PlantPalette(p *contract.PlantDesign) contract.Palette {
	var pal contract.Palette
	if n := len(p.Colors); n > 0 {
		pal.Body = p.Colors[0]
		pal.Belly = p.Colors[n/2]
		pal.Accent = p.Colors[n-1]
		pal.Glow = p.Colors[n-1]
	}
	return pal
}

// CoralPalette maps a coral's gradient colors into a Palette for the gate.
func CoralPalette(c *contract.CoralDesign) contract.Palette {
	var pal contract.Palette
	if n := len(c.Colors); n > 0 {
		pal.Body = c.Colors[0]
		pal.Belly = c.Colors[n/2]
		pal.Accent = c.Colors[n-1]
		pal.Glow = c.Colors[n-1]
	}
	return pal
}
