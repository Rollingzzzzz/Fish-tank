// G3.1: embedded core content + EnsureSeed.
//
// Seed sources live as JSON files under internal/content/seed/** and are
// compiled in via go:embed (choice documented in docs/NOTES-laneB.md). The
// runtime content/ folder is generated from them by EnsureSeed at app start.
package content

import (
	"embed"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"strings"
)

//go:embed seed
var seedFS embed.FS

// EnsureSeed writes the embedded core content (9 species, 3 water presets,
// 4 plant designs, 4 corals, 4 pattern recipes — one per stage, source "core")
// when the matching file is missing. Idempotent: existing files are never
// rewritten.
func (s *Store) EnsureSeed() error {
	return fs.WalkDir(seedFS, "seed", func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		rel := strings.TrimPrefix(p, "seed/")
		segs := strings.Split(rel, "/")
		if len(segs) != 2 || !isContentDir(segs[0]) {
			return nil
		}
		target := filepath.Join(s.root, filepath.FromSlash(rel))
		if _, err := os.Stat(target); err == nil {
			return nil // already present: keep user/agent content untouched
		}
		b, err := seedFS.ReadFile(p)
		if err != nil {
			log.Printf("content: seed read %s: %v", p, err)
			return nil
		}
		id, err := s.writeTyped(segs[0], b)
		if err != nil {
			log.Printf("content: seed %s skipped: %v", rel, err)
			return nil
		}
		log.Printf("content: seeded %s (%s)", rel, id)
		return nil
	})
}
