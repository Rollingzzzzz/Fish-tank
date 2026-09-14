// G3.1: atomic writers — clamp to a copy, validate, write tmp+rename, then
// register the C6 fingerprint in the registry.
package content

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/Rollingzzzzz/Fish-tank/internal/contract"
)

// atomicWrite writes data to a .tmp sibling then renames over the target.
func atomicWrite(path string, data []byte) error {
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

// marshalJSON produces stable, readable JSON files with a trailing newline.
func marshalJSON(v any) ([]byte, error) {
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return nil, err
	}
	return append(b, '\n'), nil
}

// defaultSource fills Source with "user" when unset (on the copy).
func defaultSource(source *string) {
	if *source == "" {
		*source = "user"
	}
}

// WriteSpecies clamps a copy of sp (role-aware N9 caps via clampSpecies —
// only the core-seeded Chosen keeps the full range; every other species is
// capped below her), validates it, writes species/<id>.json atomically and
// registers its fingerprint.
func (s *Store) WriteSpecies(sp *contract.Species) error {
	c := *sp
	clampSpecies(&c)
	defaultSource(&c.Source)
	if c.CreatedAt == "" {
		c.CreatedAt = time.Now().UTC().Format(time.RFC3339)
	}
	if err := validateSpecies(&c); err != nil {
		return err
	}
	data, err := marshalJSON(&c)
	if err != nil {
		return err
	}
	path := filepath.Join(s.path(DirSpecies), c.ID+".json")
	if err := atomicWrite(path, data); err != nil {
		return err
	}
	s.mu.Lock()
	s.upsertSpecies(&c)
	s.reg.Append(KindSpecies, c.ID, contract.Fingerprint(KindSpecies, speciesCanonical(c.Pattern, c.Palette)))
	s.mu.Unlock()
	return nil
}

// WriteWater clamps, validates, writes water/<id>.json and registers.
func (s *Store) WriteWater(w *contract.WaterPreset) error {
	c := *w
	clampWater(&c)
	defaultSource(&c.Source)
	if err := validateWater(&c); err != nil {
		return err
	}
	data, err := marshalJSON(&c)
	if err != nil {
		return err
	}
	path := filepath.Join(s.path(DirWater), c.ID+".json")
	if err := atomicWrite(path, data); err != nil {
		return err
	}
	s.mu.Lock()
	s.upsertWater(&c)
	s.reg.Append(KindWater, c.ID, contract.Fingerprint(KindWater, waterCanonical(&c)))
	s.mu.Unlock()
	return nil
}

// WritePlant clamps, validates, writes plants/<id>.json and registers.
func (s *Store) WritePlant(p *contract.PlantDesign) error {
	c := *p
	clampPlant(&c)
	defaultSource(&c.Source)
	if err := validatePlant(&c); err != nil {
		return err
	}
	data, err := marshalJSON(&c)
	if err != nil {
		return err
	}
	path := filepath.Join(s.path(DirPlants), c.ID+".json")
	if err := atomicWrite(path, data); err != nil {
		return err
	}
	s.mu.Lock()
	s.upsertPlant(&c)
	s.reg.Append(KindPlant, c.ID, contract.Fingerprint(KindPlant, plantCanonical(&c)))
	s.mu.Unlock()
	return nil
}

// WriteCoral clamps, validates, writes corals/<id>.json and registers.
func (s *Store) WriteCoral(c *contract.CoralDesign) error {
	v := *c
	clampCoral(&v)
	defaultSource(&v.Source)
	if v.CreatedAt == "" {
		v.CreatedAt = time.Now().UTC().Format(time.RFC3339)
	}
	if err := validateCoral(&v); err != nil {
		return err
	}
	data, err := marshalJSON(&v)
	if err != nil {
		return err
	}
	path := filepath.Join(s.path(DirCorals), v.ID+".json")
	if err := atomicWrite(path, data); err != nil {
		return err
	}
	s.mu.Lock()
	s.upsertCoral(&v)
	s.reg.Append(KindCoral, v.ID, contract.Fingerprint(KindCoral, coralCanonical(&v)))
	s.mu.Unlock()
	return nil
}

// WriteRecipe clamps, validates, writes patterns/<id>.json and registers the
// stage-aware recipe fingerprint (checked by GateRecipe).
func (s *Store) WriteRecipe(r *contract.PatternRecipe) error {
	c := *r
	clampRecipe(&c)
	if err := validateRecipe(&c); err != nil {
		return err
	}
	data, err := marshalJSON(&c)
	if err != nil {
		return err
	}
	path := filepath.Join(s.path(DirPatterns), c.ID+".json")
	if err := atomicWrite(path, data); err != nil {
		return err
	}
	s.mu.Lock()
	s.upsertRecipe(&c)
	s.reg.Append(KindPattern, c.ID, contract.Fingerprint(KindPattern, recipeCanonical(&c)))
	s.mu.Unlock()
	return nil
}

// writeTyped decodes data as the artifact kind stored in dir and writes it via
// the matching Write method. Returns the artifact ID. Shared by seed + import.
func (s *Store) writeTyped(dir string, data []byte) (string, error) {
	switch dir {
	case DirSpecies:
		var v contract.Species
		if err := json.Unmarshal(data, &v); err != nil {
			return "", err
		}
		return v.ID, s.WriteSpecies(&v)
	case DirWater:
		var v contract.WaterPreset
		if err := json.Unmarshal(data, &v); err != nil {
			return "", err
		}
		return v.ID, s.WriteWater(&v)
	case DirPlants:
		var v contract.PlantDesign
		if err := json.Unmarshal(data, &v); err != nil {
			return "", err
		}
		return v.ID, s.WritePlant(&v)
	case DirPatterns:
		var v contract.PatternRecipe
		if err := json.Unmarshal(data, &v); err != nil {
			return "", err
		}
		return v.ID, s.WriteRecipe(&v)
	case DirCorals:
		var v contract.CoralDesign
		if err := json.Unmarshal(data, &v); err != nil {
			return "", err
		}
		return v.ID, s.WriteCoral(&v)
	default:
		return "", fmt.Errorf("content: unknown content folder %q", dir)
	}
}

func (s *Store) upsertSpecies(v *contract.Species) {
	for i, e := range s.species {
		if e.ID == v.ID {
			s.species[i] = v
			return
		}
	}
	s.species = append(s.species, v)
}

func (s *Store) upsertWater(v *contract.WaterPreset) {
	for i, e := range s.waters {
		if e.ID == v.ID {
			s.waters[i] = v
			return
		}
	}
	s.waters = append(s.waters, v)
}

func (s *Store) upsertPlant(v *contract.PlantDesign) {
	for i, e := range s.plants {
		if e.ID == v.ID {
			s.plants[i] = v
			return
		}
	}
	s.plants = append(s.plants, v)
}

func (s *Store) upsertRecipe(v *contract.PatternRecipe) {
	for i, e := range s.recipes {
		if e.ID == v.ID {
			s.recipes[i] = v
			return
		}
	}
	s.recipes = append(s.recipes, v)
}

func (s *Store) upsertCoral(v *contract.CoralDesign) {
	for i, e := range s.corals {
		if e.ID == v.ID {
			s.corals[i] = v
			return
		}
	}
	s.corals = append(s.corals, v)
}
