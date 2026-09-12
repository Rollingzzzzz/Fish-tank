// G3.1: Store — scans, validates and serves the runtime content tree.
package content

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"

	"github.com/Rollingzzzzz/Fish-tank/internal/contract"
)

// Content sub-folders and the registry file living at the content root.
const (
	DirSpecies  = "species"
	DirWater    = "water"
	DirPlants   = "plants"
	DirPatterns = "patterns"
	DirCorals   = "corals"
	// RegistryFile is the append-only fingerprint log (C6).
	RegistryFile = "registry.json"
)

// Registry kind tags used in registry.json and by the Gate paths.
const (
	KindSpecies = "species"
	KindWater   = "water"
	KindPlant   = "plant"
	KindPattern = "pattern"
	KindCoral   = "coral"
)

// Store is the goroutine-safe view of one content folder. Create via Load.
type Store struct {
	mu      sync.Mutex
	root    string
	reg     *Registry
	species []*contract.Species
	waters  []*contract.WaterPreset
	plants  []*contract.PlantDesign
	recipes []*contract.PatternRecipe
	corals  []*contract.CoralDesign
}

// Load scans root (creating missing sub-folders) and returns a ready Store.
// Unparseable JSON, bad hex or bad IDs skip the file with a log line (D4);
// out-of-range floats are clamped, never rejected.
func Load(root string) (*Store, error) {
	if strings.TrimSpace(root) == "" {
		return nil, fmt.Errorf("content: empty root path")
	}
	s := &Store{root: root}
	for _, d := range []string{DirSpecies, DirWater, DirPlants, DirPatterns, DirCorals} {
		if err := os.MkdirAll(filepath.Join(root, d), 0o755); err != nil {
			return nil, fmt.Errorf("content: mkdir %s: %w", d, err)
		}
	}
	reg, err := LoadRegistry(filepath.Join(root, RegistryFile))
	if err != nil {
		return nil, err
	}
	s.reg = reg
	s.scanAll()
	return s, nil
}

// Root returns the content folder this store manages.
func (s *Store) Root() string { return s.root }

func (s *Store) path(dir string) string { return filepath.Join(s.root, dir) }

// scanTyped decodes every file in dir as T, clamping ranges then running the
// strict validator (bad files skipped, never fatal — D4).
func scanTyped[T any](dir string, clamp func(*T), validate func(*T) error) []*T {
	return scanDir(dir, func(b []byte) (*T, error) {
		var v T
		if err := json.Unmarshal(b, &v); err != nil {
			return nil, err
		}
		clamp(&v)
		if err := validate(&v); err != nil {
			return nil, err
		}
		return &v, nil
	})
}

func (s *Store) scanAll() {
	s.species = scanTyped(s.path(DirSpecies), clampSpecies, validateSpecies)
	s.waters = scanTyped(s.path(DirWater), clampWater, validateWater)
	s.plants = scanTyped(s.path(DirPlants), clampPlant, validatePlant)
	s.recipes = scanTyped(s.path(DirPatterns), clampRecipe, validateRecipe)
	s.corals = scanTyped(s.path(DirCorals), clampCoral, validateCoral)
}

// scanDir reads every *.json file in dir (sorted by name), decoding via fn.
// Bad files are logged and skipped, never fatal (D4).
func scanDir[T any](dir string, fn func([]byte) (*T, error)) []*T {
	entries, err := os.ReadDir(dir)
	if err != nil {
		log.Printf("content: read dir %s: %v", dir, err)
		return nil
	}
	names := make([]string, 0, len(entries))
	for _, e := range entries {
		if e.IsDir() || !strings.EqualFold(filepath.Ext(e.Name()), ".json") {
			continue
		}
		names = append(names, e.Name())
	}
	sort.Strings(names)
	out := make([]*T, 0, len(names))
	for _, n := range names {
		b, err := os.ReadFile(filepath.Join(dir, n))
		if err != nil {
			log.Printf("content: read %s: %v", n, err)
			continue
		}
		v, err := fn(b)
		if err != nil {
			log.Printf("content: skipping %s/%s: %v", filepath.Base(dir), n, err)
			continue
		}
		out = append(out, v)
	}
	return out
}

// Species returns a copy of the species list, sorted by ID.
func (s *Store) Species() []*contract.Species {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]*contract.Species, len(s.species))
	copy(out, s.species)
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out
}

// Waters returns a copy of the water preset list, sorted by ID.
func (s *Store) Waters() []*contract.WaterPreset {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]*contract.WaterPreset, len(s.waters))
	copy(out, s.waters)
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out
}

// Plants returns a copy of the plant design list, sorted by ID.
func (s *Store) Plants() []*contract.PlantDesign {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]*contract.PlantDesign, len(s.plants))
	copy(out, s.plants)
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out
}

// Recipes returns a copy of the pattern recipe list, sorted by ID.
func (s *Store) Recipes() []*contract.PatternRecipe {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]*contract.PatternRecipe, len(s.recipes))
	copy(out, s.recipes)
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out
}

// Corals returns a copy of the coral design list, sorted by ID.
func (s *Store) Corals() []*contract.CoralDesign {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]*contract.CoralDesign, len(s.corals))
	copy(out, s.corals)
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out
}

// SpeciesByID returns the species with the given ID or nil.
func (s *Store) SpeciesByID(id string) *contract.Species {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, sp := range s.species {
		if sp.ID == id {
			return sp
		}
	}
	return nil
}

// HasID reports whether an artifact with this ID exists for the kind.
func (s *Store) HasID(kind, id string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, e := range s.idNamePairs(kind) {
		if e.id == id {
			return true
		}
	}
	return false
}

// HasName reports whether an artifact with this display name (case-insensitive)
// exists for the kind.
func (s *Store) HasName(kind, name string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, e := range s.idNamePairs(kind) {
		if strings.EqualFold(e.name, name) {
			return true
		}
	}
	return false
}

// Names returns the display names of all artifacts of the kind (prompt context).
func (s *Store) Names(kind string) []string {
	s.mu.Lock()
	defer s.mu.Unlock()
	pairs := s.idNamePairs(kind)
	out := make([]string, 0, len(pairs))
	for _, e := range pairs {
		out = append(out, e.name)
	}
	return out
}

// RegistrySize returns the number of registered fingerprints (demo/log helper).
func (s *Store) RegistrySize() int { return s.reg.Len() }

type idName struct{ id, name string }

func (s *Store) idNamePairs(kind string) []idName {
	var out []idName
	switch kind {
	case KindSpecies:
		for _, e := range s.species {
			out = append(out, idName{e.ID, e.Name})
		}
	case KindWater:
		for _, e := range s.waters {
			out = append(out, idName{e.ID, e.Name})
		}
	case KindPlant:
		for _, e := range s.plants {
			out = append(out, idName{e.ID, e.Name})
		}
	case KindPattern:
		for _, e := range s.recipes {
			out = append(out, idName{e.ID, e.Name})
		}
	case KindCoral:
		for _, e := range s.corals {
			out = append(out, idName{e.ID, e.Name})
		}
	}
	return out
}
