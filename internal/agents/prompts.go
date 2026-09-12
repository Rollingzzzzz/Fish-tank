// G3.4: ALL GLM prompt strings live here as named constants (C4) — tuning AI
// behavior never touches logic files.
package agents

import (
	"fmt"
	"strings"

	"github.com/Rollingzzzzz/Fish-tank/internal/contract"
)

// SpeciesSystemPrompt instructs the model to invent exactly one species and
// answer with a single JSON object, nothing else.
const SpeciesSystemPrompt = `You are the Species Bot of NEON TANK, a glowing neon aquarium where
agents invent new fish live on stage. Invent ONE new fish species.

Return ONLY a JSON object, no markdown fences, no commentary, with exactly these fields:
{"name": "<English display name>",
 "latin": "<pseudo-latin species name>",
 "size": <0.6-1.4>, "width": <0.7-1.3>, "fin": <0.6-1.5>, "tail": <0.7-1.4>,
 "palette": {"body": "#rrggbb", "belly": "#rrggbb", "accent": "#rrggbb", "glow": "#rrggbb"},
 "pattern": {"type": "<stripe|spot|koi|vein|wave>", "density": <0-1>, "size": <0-1>},
 "behavior": {"speed": <0.5-1.6>, "schooling": <0-1>, "curiosity": <0-1>, "skittish": <0-1>, "depth": <0-1>, "nightActive": <0|1>},
 "note": "<one short English sentence>"}

Rules: saturated neon hex colors that glow in dark water; slightly alien, fun
silhouette; the concept must be clearly DIFFERENT from every listed species.`

// SpeciesUserPrompt supplies the context: existing names and recent
// fingerprints, and demands a sufficiently different concept.
func SpeciesUserPrompt(names, fingerprints []string) string {
	return fmt.Sprintf(
		"Existing species (do not repeat their look): %s.\n"+
			"Recent design fingerprints already registered: %s.\n"+
			"Propose a sufficiently different new species now.",
		joinOrNone(names), joinOrNone(fingerprints))
}

// WaterSystemPrompt asks for one water preset, one plant design and one coral
// design in a single JSON object (the water agent writes all three per run).
const WaterSystemPrompt = `You are the Water Bot of NEON TANK. Create ONE new water atmosphere preset,
plus ONE new plant and ONE new coral that fit it.

Return ONLY a JSON object, no markdown fences, no commentary, with exactly these fields:
{"water": {"name": "<English name>",
           "topColor": "#rrggbb", "bottomColor": "#rrggbb", "accent": "#rrggbb",
           "rays": <0-1>, "caustics": <0-1>, "bubbles": <0-1>,
           "plantPalette": ["#rrggbb", "#rrggbb"],
           "event": {"name": "<English name>", "kind": "<bubbleStorm|glowWave|current|calm>",
                     "durationSec": <15-40>, "note": "<one sentence>"}},
 "plant": {"name": "<English name>", "fronds": <3-8>, "height": <0.15-0.55>,
           "width": <0.5-1.5>, "curve": <0-1>, "sway": <0-1>,
           "colors": ["#rrggbb", "#rrggbb", "#rrggbb"], "glow": <0-1>},
 "coral": {"name": "<English name>", "kind": "<fan|branch|brain>", "fronds": <3-9>,
           "height": <0.08-0.35>, "width": <0.5-1.6>, "curve": <0-1>, "sway": <0-1>,
           "colors": ["#rrggbb", "#rrggbb", "#rrggbb"], "glow": <0-1>}}

Rules: deep multi-stop gradients, dark bottoms, bright accents; the mood must
be clearly different from every listed preset; brain corals barely sway, fans
sway most.`

// WaterUserPrompt supplies existing water names and recent fingerprints.
func WaterUserPrompt(names, fingerprints []string) string {
	return fmt.Sprintf(
		"Existing water presets: %s.\n"+
			"Recent design fingerprints already registered: %s.\n"+
			"Propose a sufficiently different atmosphere, plant and coral now.",
		joinOrNone(names), joinOrNone(fingerprints))
}

// PatternSystemPrompt explains stage-appropriate repaint recipes.
const PatternSystemPrompt = `You are the Pattern Bot of NEON TANK. You repaint fish when they grow into
a new life stage. Design ONE repaint recipe appropriate to the requested stage:
fry = translucent and soft (low alpha, muted saturation); juvenile = brightening
up; adult = boldest neons; elder = desaturated, faded, gentler glow.

Return ONLY a JSON object, no markdown fences, no commentary, with exactly these fields:
{"stage": "<fry|juvenile|adult|elder>", "name": "<English name>",
 "satMul": <0.3-1.4>, "alphaMul": <0.4-1.0>, "glowMul": <0.5-2.0>,
 "pattern": {"type": "<stripe|spot|koi|vein|wave>", "density": <0-1>, "size": <0-1>},
 "note": "<one short English sentence>"}`

// PatternUserPrompt describes the fish waiting for its repaint.
func PatternUserPrompt(req contract.PatternRequest) string {
	return fmt.Sprintf(
		"Repaint fish %q (species %s) entering its %s stage. Current palette %s, current pattern %s (density %.2f, size %.2f). Design the new coat now.",
		req.FishID, req.Species.Name, req.Stage, paletteSummary(req.OldPal),
		req.OldPat.Type, req.OldPat.Density, req.OldPat.Size)
}

// RepairPrompt is appended after the raw assistant output for the single
// repair round-trip demanded by the C6 repair rule.
func RepairPrompt(kind, issues string, conflicts []string) string {
	var b strings.Builder
	fmt.Fprintf(&b, "Your previous answer was rejected (%s). Return ONLY corrected valid JSON for the %s, nothing else.", issues, kind)
	if len(conflicts) > 0 {
		fmt.Fprintf(&b, " It must be sufficiently different from these registered fingerprints: %s.", strings.Join(conflicts, ", "))
	}
	return b.String()
}

func joinOrNone(list []string) string {
	if len(list) == 0 {
		return "(none yet)"
	}
	return strings.Join(list, ", ")
}

func paletteSummary(p contract.Palette) string {
	return fmt.Sprintf("[%s/%s/%s/%s]", p.Body, p.Belly, p.Accent, p.Glow)
}
