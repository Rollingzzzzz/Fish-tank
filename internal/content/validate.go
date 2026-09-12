// G3.1: strict validation (slug/hex/enum) plus section 4 range clamping.
// Ranges are clamped on load and write, never rejected (D4); only bad slugs,
// bad hex colors and unknown enum values are validation errors.
package content

import (
	"fmt"
	"strings"

	"github.com/Rollingzzzzz/Fish-tank/internal/contract"
)

// README section 4 numeric ranges (C4: documented tunables; clamped, never fatal).
const (
	spSizeMin, spSizeMax       = 0.6, 1.4
	spWidthMin, spWidthMax     = 0.7, 1.3
	spFinMin, spFinMax         = 0.6, 1.5
	spTailMin, spTailMax       = 0.7, 1.4
	spSpeedMin, spSpeedMax     = 0.5, 1.6
	unitMin, unitMax           = 0.0, 1.0
	wEventDurMin, wEventDurMax = 15.0, 40.0
	plHeightMin, plHeightMax   = 0.15, 0.55
	plWidthMin, plWidthMax     = 0.5, 1.5
	plFrondsMin, plFrondsMax   = 3, 8
	coHeightMin, coHeightMax   = 0.08, 0.35
	coWidthMin, coWidthMax     = 0.5, 1.6
	coFrondsMin, coFrondsMax   = 3, 9
	rcSatMin, rcSatMax         = 0.3, 1.4
	rcAlphaMin, rcAlphaMax     = 0.4, 1.0
	rcGlowMin, rcGlowMax       = 0.5, 2.0
)

// Frozen enum sets.
var (
	patternTypes = map[string]bool{"stripe": true, "spot": true, "koi": true, "vein": true, "wave": true}
	eventKinds   = map[string]bool{"bubbleStorm": true, "glowWave": true, "current": true, "calm": true}
	stageNames   = map[string]bool{"fry": true, "juvenile": true, "adult": true, "elder": true}
	coralKinds   = map[string]bool{"fan": true, "branch": true, "brain": true}
	plantKinds   = map[string]bool{"ribbon": true, "feather": true, "silk": true} // v0.3.7 F30
	speciesRoles = map[string]bool{contract.RoleNormal: true, contract.RoleChosen: true}
)

// validID reports whether id matches ^[a-z0-9-]{3,32}$.
func validID(id string) bool {
	if len(id) < 3 || len(id) > 32 {
		return false
	}
	for _, r := range id {
		if (r < 'a' || r > 'z') && (r < '0' || r > '9') && r != '-' {
			return false
		}
	}
	return true
}

// clampColors truncates/pads a hex color slice into the [min,max] range by
// repeating the last color (graceful degradation, D4).
func clampColors(hexes []string, min, max int) []string {
	out := append([]string(nil), hexes...)
	if len(out) > max {
		out = out[:max]
	}
	for len(out) < min && len(out) > 0 {
		out = append(out, out[len(out)-1])
	}
	for len(out) < min {
		out = append(out, "#808080")
	}
	return out
}

// clampSpecies clamps README §4 ranges, role-aware (N9/F23): the full
// 0.6..1.4 / 0.6..1.5 / 0.7..1.4 / 0.5..1.6 ranges stay reserved for the
// core-seeded Chosen alone; every normal species (and any non-core "chosen")
// is capped below her via contract.NormalSizeMax/FinMax/TailMax/SpeedMax so
// no fish ever matches her scale, finery — or pace.
func clampSpecies(sp *contract.Species) {
	if sp.Role == "" {
		sp.Role = contract.RoleNormal // legacy v1 files have no role field
	}
	sizeMax, finMax, tailMax, speedMax := spSizeMax, spFinMax, spTailMax, spSpeedMax
	if sp.Role != contract.RoleChosen || sp.Source != "core" {
		sizeMax, finMax, tailMax = contract.NormalSizeMax, contract.NormalFinMax, contract.NormalTailMax
		speedMax = contract.NormalSpeedMax
	}
	sp.Size = contract.Clamp(sp.Size, spSizeMin, sizeMax)
	sp.Width = contract.Clamp(sp.Width, spWidthMin, spWidthMax)
	sp.Fin = contract.Clamp(sp.Fin, spFinMin, finMax)
	sp.Tail = contract.Clamp(sp.Tail, spTailMin, tailMax)
	sp.Pattern.Density = contract.Clamp(sp.Pattern.Density, unitMin, unitMax)
	sp.Pattern.Size = contract.Clamp(sp.Pattern.Size, unitMin, unitMax)
	b := &sp.Behavior
	b.Speed = contract.Clamp(b.Speed, spSpeedMin, speedMax)
	b.Schooling = contract.Clamp(b.Schooling, unitMin, unitMax)
	b.Curiosity = contract.Clamp(b.Curiosity, unitMin, unitMax)
	b.Skittish = contract.Clamp(b.Skittish, unitMin, unitMax)
	b.Depth = contract.Clamp(b.Depth, unitMin, unitMax)
	b.Attachment = contract.Clamp(b.Attachment, unitMin, unitMax)
}

func clampWater(w *contract.WaterPreset) {
	w.Rays = contract.Clamp(w.Rays, unitMin, unitMax)
	w.Caustics = contract.Clamp(w.Caustics, unitMin, unitMax)
	w.Bubbles = contract.Clamp(w.Bubbles, unitMin, unitMax)
	w.PlantPalette = clampColors(w.PlantPalette, 2, 4)
	if w.Event != nil {
		w.Event.Duration = contract.Clamp(w.Event.Duration, wEventDurMin, wEventDurMax)
	}
}

func clampPlant(p *contract.PlantDesign) {
	if !plantKinds[p.Kind] {
		p.Kind = "ribbon" // legacy/lazy input keeps the classic silhouette
	}
	p.Fronds = int(contract.Clamp(float64(p.Fronds), plFrondsMin, plFrondsMax))
	p.Height = contract.Clamp(p.Height, plHeightMin, plHeightMax)
	p.Width = contract.Clamp(p.Width, plWidthMin, plWidthMax)
	p.Curve = contract.Clamp(p.Curve, unitMin, unitMax)
	p.Sway = contract.Clamp(p.Sway, unitMin, unitMax)
	p.Glow = contract.Clamp(p.Glow, unitMin, unitMax)
	p.Colors = clampColors(p.Colors, 2, 4)
}

func clampRecipe(r *contract.PatternRecipe) {
	r.SatMul = contract.Clamp(r.SatMul, rcSatMin, rcSatMax)
	r.AlphaMul = contract.Clamp(r.AlphaMul, rcAlphaMin, rcAlphaMax)
	r.GlowMul = contract.Clamp(r.GlowMul, rcGlowMin, rcGlowMax)
	r.Pattern.Density = contract.Clamp(r.Pattern.Density, unitMin, unitMax)
	r.Pattern.Size = contract.Clamp(r.Pattern.Size, unitMin, unitMax)
}

func clampCoral(c *contract.CoralDesign) {
	if c.Kind == "" {
		c.Kind = "fan" // lazy/legacy input defaults to the plainest look
	}
	c.Fronds = int(contract.Clamp(float64(c.Fronds), coFrondsMin, coFrondsMax))
	c.Height = contract.Clamp(c.Height, coHeightMin, coHeightMax)
	c.Width = contract.Clamp(c.Width, coWidthMin, coWidthMax)
	c.Curve = contract.Clamp(c.Curve, unitMin, unitMax)
	c.Sway = contract.Clamp(c.Sway, unitMin, unitMax)
	c.Glow = contract.Clamp(c.Glow, unitMin, unitMax)
	c.Colors = clampColors(c.Colors, 2, 4)
}

func validatePalette(p contract.Palette, what string) error {
	fields := []struct{ name, hex string }{
		{"body", p.Body}, {"belly", p.Belly}, {"accent", p.Accent}, {"glow", p.Glow},
	}
	for _, f := range fields {
		if !contract.ValidHex(f.hex) {
			return fmt.Errorf("%s palette %s %q is not #rrggbb", what, f.name, f.hex)
		}
	}
	return nil
}

func validateSpecies(sp *contract.Species) error {
	if !validID(sp.ID) {
		return fmt.Errorf("species id %q is not a slug (^[a-z0-9-]{3,32}$)", sp.ID)
	}
	if strings.TrimSpace(sp.Name) == "" {
		return fmt.Errorf("species %s has an empty name", sp.ID)
	}
	if !speciesRoles[sp.Role] {
		return fmt.Errorf("species %s: unknown role %q", sp.ID, sp.Role)
	}
	if err := validatePalette(sp.Palette, "species "+sp.ID); err != nil {
		return err
	}
	if !patternTypes[sp.Pattern.Type] {
		return fmt.Errorf("species %s: unknown pattern type %q", sp.ID, sp.Pattern.Type)
	}
	return nil
}

func validateWater(w *contract.WaterPreset) error {
	if !validID(w.ID) {
		return fmt.Errorf("water id %q is not a slug", w.ID)
	}
	if strings.TrimSpace(w.Name) == "" {
		return fmt.Errorf("water %s has an empty name", w.ID)
	}
	for name, hex := range map[string]string{"topColor": w.TopColor, "bottomColor": w.BottomColor, "accent": w.Accent} {
		if !contract.ValidHex(hex) {
			return fmt.Errorf("water %s: %s %q is not #rrggbb", w.ID, name, hex)
		}
	}
	for _, hex := range w.PlantPalette {
		if !contract.ValidHex(hex) {
			return fmt.Errorf("water %s: plantPalette entry %q is not #rrggbb", w.ID, hex)
		}
	}
	if w.Event != nil && !eventKinds[w.Event.Kind] {
		return fmt.Errorf("water %s: unknown event kind %q", w.ID, w.Event.Kind)
	}
	return nil
}

func validatePlant(p *contract.PlantDesign) error {
	if !validID(p.ID) {
		return fmt.Errorf("plant id %q is not a slug", p.ID)
	}
	if strings.TrimSpace(p.Name) == "" {
		return fmt.Errorf("plant %s has an empty name", p.ID)
	}
	for _, hex := range p.Colors {
		if !contract.ValidHex(hex) {
			return fmt.Errorf("plant %s: color %q is not #rrggbb", p.ID, hex)
		}
	}
	return nil
}

func validateCoral(c *contract.CoralDesign) error {
	if !validID(c.ID) {
		return fmt.Errorf("coral id %q is not a slug", c.ID)
	}
	if strings.TrimSpace(c.Name) == "" {
		return fmt.Errorf("coral %s has an empty name", c.ID)
	}
	if !coralKinds[c.Kind] {
		return fmt.Errorf("coral %s: unknown kind %q", c.ID, c.Kind)
	}
	for _, hex := range c.Colors {
		if !contract.ValidHex(hex) {
			return fmt.Errorf("coral %s: color %q is not #rrggbb", c.ID, hex)
		}
	}
	return nil
}

func validateRecipe(r *contract.PatternRecipe) error {
	if !validID(r.ID) {
		return fmt.Errorf("recipe id %q is not a slug", r.ID)
	}
	if strings.TrimSpace(r.Name) == "" {
		return fmt.Errorf("recipe %s has an empty name", r.ID)
	}
	if !stageNames[r.Stage] {
		return fmt.Errorf("recipe %s: unknown stage %q", r.ID, r.Stage)
	}
	if !patternTypes[r.Pattern.Type] {
		return fmt.Errorf("recipe %s: unknown pattern type %q", r.ID, r.Pattern.Type)
	}
	return nil
}

// ValidateSpecies strictly checks slug ID, non-empty name, hex colors and the
// pattern type enum. Ranges are NOT checked here (they are clamped on write).
func (s *Store) ValidateSpecies(sp *contract.Species) error { return validateSpecies(sp) }

// ValidateWater strictly checks slug ID, name, hex colors and event kind.
func (s *Store) ValidateWater(w *contract.WaterPreset) error { return validateWater(w) }

// ValidatePlant strictly checks slug ID, name and hex colors.
func (s *Store) ValidatePlant(p *contract.PlantDesign) error { return validatePlant(p) }

// ValidateCoral strictly checks slug ID, name, kind enum and hex colors.
func (s *Store) ValidateCoral(c *contract.CoralDesign) error { return validateCoral(c) }

// ValidateRecipe strictly checks slug ID, name, stage and pattern type.
func (s *Store) ValidateRecipe(r *contract.PatternRecipe) error { return validateRecipe(r) }
