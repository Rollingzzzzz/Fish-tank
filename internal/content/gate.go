// G3.6: acceptance gate — C6 fingerprint uniqueness plus case-insensitive
// display-name uniqueness, consulted by every agent before writing.
package content

import (
	"fmt"
	"strings"

	"github.com/Rollingzzzzz/Fish-tank/internal/contract"
)

// Gate checks whether a new artifact (species/water/plant kind) is acceptable:
// its display name must be unique case-insensitively within its kind, and its
// C6 fingerprint must be absent from the registry. On rejection, conflicts
// lists the colliding registry hash(es) (prefixed "name:" for name clashes) so
// the calling agent can feed them into a repair round-trip.
func (s *Store) Gate(kind, id, name string, pat contract.Pattern, pal contract.Palette) (ok bool, conflicts []string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Case-insensitive name uniqueness within the kind (C6), excluding self.
	for _, e := range s.idNamePairs(kind) {
		if e.id != id && strings.EqualFold(e.name, name) {
			conflicts = append(conflicts, "name:"+strings.ToLower(name))
			break
		}
	}

	// Fingerprint uniqueness against the append-only registry.
	fp := contract.Fingerprint(kind, canonicalFor(kind, pat, pal))
	if s.reg.Has(fp) {
		conflicts = append(conflicts, fp)
	}
	if len(conflicts) > 0 {
		return false, conflicts
	}
	return true, nil
}

// GateRecipe is the recipe-specific variant of Gate: recipe fingerprints cover
// stage and the stage multipliers instead of a palette (see recipeCanonical).
func (s *Store) GateRecipe(r *contract.PatternRecipe) (ok bool, conflicts []string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	for _, e := range s.idNamePairs(KindPattern) {
		if e.id != r.ID && strings.EqualFold(e.name, r.Name) {
			conflicts = append(conflicts, "name:"+strings.ToLower(r.Name))
			break
		}
	}

	fp := contract.Fingerprint(KindPattern, recipeCanonical(r))
	if s.reg.Has(fp) {
		conflicts = append(conflicts, fp)
	}
	if len(conflicts) > 0 {
		return false, conflicts
	}
	return true, nil
}

// GateCoral is the coral variant of Gate: name uniqueness plus the kind-aware
// coral fingerprint (kind + fronds + curve/sway + gradient hues).
func (s *Store) GateCoral(c *contract.CoralDesign) (ok bool, conflicts []string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	for _, e := range s.idNamePairs(KindCoral) {
		if e.id != c.ID && strings.EqualFold(e.name, c.Name) {
			conflicts = append(conflicts, "name:"+strings.ToLower(c.Name))
			break
		}
	}
	fp := contract.Fingerprint(KindCoral, coralCanonical(c))
	if s.reg.Has(fp) {
		conflicts = append(conflicts, fp)
	}
	if len(conflicts) > 0 {
		return false, conflicts
	}
	return true, nil
}

// chosenSpeciesID is the one exempt holder of the reserved lilac band (FD8).
const chosenSpeciesID = "chosen-lilastar"

// lilacReserved reports whether any palette hue falls inside the reserved
// lilac band [270,300] degrees (FD8: those hues belong to the Chosen alone).
func lilacReserved(pal contract.Palette) bool {
	for _, h := range []float64{
		hexHueDeg(pal.Body), hexHueDeg(pal.Belly),
		hexHueDeg(pal.Accent), hexHueDeg(pal.Glow),
	} {
		if h >= 270 && h <= 300 {
			return true
		}
	}
	return false
}

// GateSpecies is the species gate path: the generic C6 checks plus the FD8
// lilac reservation. Core-seeded artifacts and the Chosen herself are exempt;
// anyone else wearing lilac is rejected so the agent repairs the design.
func (s *Store) GateSpecies(sp *contract.Species) (ok bool, conflicts []string) {
	if ok, conflicts = s.Gate(KindSpecies, sp.ID, sp.Name, sp.Pattern, sp.Palette); !ok {
		return false, conflicts
	}
	if sp.ID != chosenSpeciesID && sp.Source != "core" && lilacReserved(sp.Palette) {
		return false, append(conflicts, "lilac:reserved-270-300")
	}
	return true, nil
}

// RecentFingerprints returns up to n registry hashes, newest last; agents pass
// them to the LLM as "avoid these designs" context.
func (s *Store) RecentFingerprints(n int) []string { return s.reg.Recent(n) }

// canonicalFor dispatches the canonical builder for the generic Gate and must
// stay identical to what the matching Write* method registers. Water and plant
// canonicals are hue-only (exactly like the write path); recipes carry their
// multipliers and therefore use GateRecipe instead.
func canonicalFor(kind string, pat contract.Pattern, pal contract.Palette) string {
	switch kind {
	case KindWater:
		return hueCanonical("water", pal)
	case KindPlant:
		return hueCanonical("plant", pal)
	default: // KindSpecies
		return speciesCanonical(pat, pal)
	}
}

func itoa(i int) string { return fmt.Sprintf("%d", i) }
