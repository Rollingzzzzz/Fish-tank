// G3.3: simulate helpers — HSL to hex (own implementation, stdlib only),
// neon palette random walks, English name syllable combiner, note templates
// and unique ID helpers. All generators are deterministic per *rand.Rand (D5).
package agents

import (
	"fmt"
	"math"
	"math/rand"
	"strings"

	"github.com/Rollingzzzzz/Fish-tank/internal/content"
	"github.com/Rollingzzzzz/Fish-tank/internal/contract"
)

// hslHex converts hue [0,360), saturation/lightness [0,1] to "#rrggbb".
func hslHex(h, s, l float64) string {
	h = math.Mod(math.Mod(h, 360)+360, 360)
	c := (1 - math.Abs(2*l-1)) * s
	x := c * (1 - math.Abs(math.Mod(h/60, 2)-1))
	m := l - c/2
	var r, g, b float64
	switch {
	case h < 60:
		r, g, b = c, x, 0
	case h < 120:
		r, g, b = x, c, 0
	case h < 180:
		r, g, b = 0, c, x
	case h < 240:
		r, g, b = 0, x, c
	case h < 300:
		r, g, b = x, 0, c
	default:
		r, g, b = c, 0, x
	}
	ch := func(v float64) int { return int(math.Round((v + m) * 255)) }
	return fmt.Sprintf("#%02x%02x%02x", ch(r), ch(g), ch(b))
}

// neonPaletteAt builds a saturated neon palette as a random walk around hue
// (art direction section 3: glow first, multi-stop gradient feel).
func neonPaletteAt(rng *rand.Rand, hue float64) contract.Palette {
	body := hslHex(hue, 0.85+rng.Float64()*0.15, 0.5+rng.Float64()*0.12)
	belly := hslHex(hue+20+rng.Float64()*30, 0.55+rng.Float64()*0.3, 0.66+rng.Float64()*0.14)
	accentHue := hue + 150 + rng.Float64()*60
	accent := hslHex(accentHue, 0.9+rng.Float64()*0.1, 0.52+rng.Float64()*0.1)
	glow := hslHex(accentHue+8+rng.Float64()*14, 1.0, 0.58+rng.Float64()*0.08)
	return contract.Palette{Body: body, Belly: belly, Accent: accent, Glow: glow}
}

// gradientColors walks the hue in n stops from deep base to glowing tip.
func gradientColors(rng *rand.Rand, hue float64, n int) []string {
	out := make([]string, 0, n)
	for i := 0; i < n; i++ {
		h := hue + float64(i)*14
		sat := 0.8 + 0.2*float64(i)/float64(n)
		light := 0.32 + 0.45*float64(i)/float64(n-1)
		out = append(out, hslHex(h, sat, light))
	}
	return out
}

func pick(rng *rand.Rand, list []string) string { return list[rng.Intn(len(list))] }

// Name syllable pools (English display names, D6).
var (
	namePrefix = []string{"Neon", "Volt", "Lumen", "Aqua", "Cryo", "Pyro", "Umbra", "Astra", "Zephyr", "Nova", "Ember", "Glimmer", "Prism", "Echo"}
	nameCore   = []string{"Ember", "Coral", "Violet", "Ghost", "Comet", "Mirror", "Static", "Aurora", "Ripple", "Halo", "Onyx", "Solar"}
	nameTail   = []string{"fin", "tail", "drift", "glow", "dart", "shade", "wing", "flare", "gleam", "veil"}
	latinGenus = []string{"Lampyris", "Cyanogaster", "Violaceus", "Aureolus", "Noctalis", "Photia", "Spectrus", "Miralus", "Fulgorus", "Iridessa"}
	latinEpith = []string{"ignis", "velum", "lucens", "umbra", "caelestis", "mirus", "aurora", "profundus", "electricus", "serenus"}

	waterCore = []string{"Aurora", "Midnight", "Neon", "Crystal", "Prism", "Twilight", "Solar", "Lunar", "Violet", "Emerald"}
	waterTail = []string{"Trench", "Lagoon", "Basin", "Reef", "Depths", "Shallows", "Cove", "Current"}

	plantCore = []string{"Glow", "Ribbon", "Jelly", "Bubble", "Spiral", "Crystal", "Ember", "Violet", "Frost", "Pulse"}
	plantTail = []string{"fern", "grass", "tendril", "moss", "palm", "reed", "bloom", "vine", "kelp", "plume"}

	coralCore     = []string{"Ember", "Violet", "Crystal", "Rose", "Cyan", "Glimmer", "Aurora", "Onyx"}
	coralTail     = []string{"Seafan", "Candle", "Boulder", "Veil", "Crown", "Torch", "Bubble", "Crest"}
	coralKindPool = []string{"fan", "branch", "brain"}

	eventKindPool = []string{"bubbleStorm", "glowWave", "current", "calm"}
	patternPool   = []string{"stripe", "spot", "koi", "vein", "wave"}
	stagePool     = []string{"fry", "juvenile", "adult", "elder"}

	speciesNotes = []string{
		"%s glides through mid-water trailing flickering light.",
		"%s darts between the plants in sudden neon bursts.",
		"%s hovers near the glass and watches everything move.",
		"%s sweeps the open water in slow, glowing arcs.",
		"%s pulses brighter whenever the school changes course.",
	}
	plantNotes = []string{
		"Fronds bend slowly like a living ribbon in the current.",
		"Tips pulse with a soft glow after lights-out.",
		"Curled fronds catch drifting bubbles on their edges.",
		"Glows warmly at night, anchoring the mid-ground.",
	}
	waterNotes = []string{
		"A slow mood that makes every neon stripe pop.",
		"Deep water with a bright electric signature.",
		"Soft drifting light, made for lazy schools.",
		"Charged atmosphere with a cold metallic shine.",
	}
	recipeNotes = map[string]string{
		"fry":      "Translucent and soft so the little body glows through.",
		"juvenile": "First bold colors coming in while schooling improves.",
		"adult":    "Peak neon: boldest saturation of the whole life.",
		"elder":    "Desaturated and faded, a gentle dusk glow remains.",
	}
)

// slugWithHeadroom keeps room for numeric suffixes within the 32-char limit.
func slugWithHeadroom(name string) string {
	base := contract.Slugify(name)
	if len(base) > 28 {
		base = strings.TrimRight(base[:28], "-")
	}
	return base
}

// uniqueID returns base, or base-2, base-3 ... until free in the store.
func uniqueID(st *content.Store, kind, base string) string {
	id := base
	for n := 2; st.HasID(kind, id); n++ {
		id = fmt.Sprintf("%s-%d", base, n)
		if len(id) > 32 {
			id = id[:32]
		}
	}
	return id
}

// uniqueName regenerates until the display name is free (case-insensitive),
// widening the space with a numeric suffix when the pools run dry.
func uniqueName(rng *rand.Rand, st *content.Store, kind string, gen func() string) string {
	for i := 0; i < 64; i++ {
		name := gen()
		if i > 32 {
			name = fmt.Sprintf("%s %d", name, i)
		}
		if !st.HasName(kind, name) {
			return name
		}
	}
	return fmt.Sprintf("Drifter %d", rng.Intn(9000)+1000)
}

// RandomStage picks one of the four life stages.
func RandomStage(rng *rand.Rand) string { return pick(rng, stagePool) }
