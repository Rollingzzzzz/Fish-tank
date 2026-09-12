// G0.2: contract sanity tests.
package contract

import (
	"math"
	"strings"
	"testing"
)

func TestClamp(t *testing.T) {
	if Clamp(-1, 0, 1) != 0 || Clamp(2, 0, 1) != 1 || Clamp(0.5, 0, 1) != 0.5 {
		t.Fatal("Clamp out-of-range behavior wrong")
	}
}

func TestFingerprintDeterministic(t *testing.T) {
	a := Fingerprint("species", "stripe|d0.35|s0.50|h120|h300")
	b := Fingerprint("species", "stripe|d0.35|s0.50|h120|h300")
	c := Fingerprint("species", "stripe|d0.40|s0.50|h120|h300")
	if a != b || len(a) != 16 {
		t.Fatalf("fingerprint not deterministic/16hex: %q vs %q", a, b)
	}
	if a == c {
		t.Fatal("different canonicals must produce different fingerprints")
	}
}

func TestSlugify(t *testing.T) {
	cases := map[string]bool{
		Slugify("Neon Vora"):   true,
		Slugify("  --A B--  "): true,
		Slugify("Ünïcode!"):    true,
		Slugify("ab"):          true, // padded to 3
	}
	for s, ok := range cases {
		if len(s) < 3 || len(s) > 32 {
			t.Fatalf("slug %q length out of range", s)
		}
		for _, r := range s {
			if !strings.ContainsRune("abcdefghijklmnopqrstuvwxyz0123456789-", r) {
				t.Fatalf("slug %q has illegal rune %q", s, r)
			}
		}
		if !ok {
			t.Fatal("unreachable")
		}
	}
}

func TestQuantizeAndHex(t *testing.T) {
	q := Quantize(0.37, 0.05)
	if math.Abs(q-0.35) > 1e-9 {
		t.Fatalf("Quantize wrong: %v", q)
	}
	if !ValidHex("#00ff88") || ValidHex("00ff88") || ValidHex("#00ff8") {
		t.Fatal("ValidHex wrong")
	}
	r, g, b := HexToRGB("#ff8001")
	if r != 0xff || g != 0x80 || b != 0x01 {
		t.Fatalf("HexToRGB wrong: %v %v %v", r, g, b)
	}
}

func TestTuningFrozen(t *testing.T) {
	if SpineSegments != 14 || SaveSchemaVersion != 2 || MaxOutputTokens != 1600 {
		t.Fatal("tuning constants drifted from README §4/§10")
	}
	if AgentSchedules["species"] != 140 || AgentFirstRun["water"] != 6 {
		t.Fatal("agent schedule constants drifted")
	}
	if MinPopulation != 4 || ZoneRadius != 140 || MusicLoopSec != 64 || TreatMax != 6 {
		t.Fatal("v0.2 tuning constants drifted from README §10")
	}
	// v0.3.6 (F23/F25): supremacy speeds and the volcanic floor split are frozen
	if NormalSpeedMax != 1.35 || ChosenSpeedMul != 1.45 ||
		FloorShareRocks != 0.50 || FloorSharePlants != 0.15 || FloorShareCorals != 0.15 {
		t.Fatal("v0.3.6 tuning constants drifted from README §11")
	}
	// v0.3.8: TransitHideSec retired (binary swallow); the carry cap stays frozen
	if HeldScentGrowCap != 140 {
		t.Fatal("v0.3.6 carry constants drifted")
	}
	// v0.3.7 (F29): the player may stock up to 100 fish.
	if PopCapMax != 100 {
		t.Fatal("v0.3.7 PopCapMax drifted from README §11")
	}
}
