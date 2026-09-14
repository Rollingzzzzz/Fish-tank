// F30: the Kind field — legacy/unknown kinds fall back to the classic
// ribbon, the new feather/silk archetypes survive validation, and the six
// hand-authored silky seeds ship in the content store.
package content

import (
	"testing"

	"github.com/Rollingzzzzz/Fish-tank/internal/contract"
)

func TestPlantKindClamp(t *testing.T) {
	p := contract.PlantDesign{ID: "test-plant", Name: "Test Plant", Fronds: 4,
		Height: 0.3, Width: 1, Colors: []string{"#123456", "#654321"}}
	clampPlant(&p)
	if p.Kind != "ribbon" {
		t.Fatalf("empty kind must default to ribbon, got %q", p.Kind)
	}
	for _, k := range []string{"feather", "silk"} {
		p.Kind = k
		clampPlant(&p)
		if p.Kind != k {
			t.Fatalf("kind %q must survive the clamp, got %q", k, p.Kind)
		}
	}
	p.Kind = "neon-tube"
	clampPlant(&p)
	if p.Kind != "ribbon" {
		t.Fatalf("unknown kind must fall back to ribbon, got %q", p.Kind)
	}
}

func TestSilkySeedsShipInStore(t *testing.T) {
	dir := t.TempDir()
	st, err := Load(dir) // seeds into the empty store
	if err != nil {
		t.Fatal(err)
	}
	if err := st.EnsureSeed(); err != nil {
		t.Fatal(err)
	}
	want := map[string]string{
		"ghost-pen-feather":      "feather",
		"ember-plume-feather":    "feather",
		"twilight-plume-feather": "feather",
		"moon-silk-grass":        "silk",
		"abyss-silkthread":       "silk",
		"pearl-veil-silk":        "silk",
	}
	found := 0
	for _, pl := range st.Plants() {
		if k, ok := want[pl.ID]; ok {
			found++
			if pl.Kind != k {
				t.Fatalf("%s loaded with kind %q, want %q", pl.ID, pl.Kind, k)
			}
		}
	}
	if found != len(want) {
		t.Fatalf("only %d of %d silky seeds shipped in the store", found, len(want))
	}
}
