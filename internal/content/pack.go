// G3.1: pack export/import — zip the content tree, extract+validate into it,
// merging registry.json deduped by hash (C6).
package content

import (
	"archive/zip"
	"io"
	"io/fs"
	"log"
	"os"
	"path"
	"path/filepath"
	"strings"
)

// ExportPack zips the whole content tree (species/ water/ plants/ patterns/ +
// registry.json) to zipPath. Temporary .tmp files are skipped.
func (s *Store) ExportPack(zipPath string) error {
	f, err := os.Create(zipPath)
	if err != nil {
		return err
	}
	defer f.Close()
	zw := zip.NewWriter(f)
	walkErr := filepath.WalkDir(s.root, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		rel, err := filepath.Rel(s.root, p)
		if err != nil {
			return err
		}
		rel = filepath.ToSlash(rel)
		if strings.HasSuffix(rel, ".tmp") {
			return nil
		}
		w, err := zw.Create(rel)
		if err != nil {
			return err
		}
		src, err := os.Open(p)
		if err != nil {
			return err
		}
		defer src.Close()
		_, err = io.Copy(w, src)
		return err
	})
	if walkErr != nil {
		zw.Close()
		return walkErr
	}
	return zw.Close()
}

// ImportPack extracts a pack into the store: every content JSON is validated
// (and clamped) before being written via the matching Write method; the pack's
// registry.json is merged deduped by hash. Invalid entries are logged and
// skipped, never fatal (D4).
func (s *Store) ImportPack(zipPath string) error {
	r, err := zip.OpenReader(zipPath)
	if err != nil {
		return err
	}
	defer r.Close()

	for _, f := range r.File {
		name := path.Clean(filepath.ToSlash(f.Name))
		if strings.HasPrefix(name, "../") || strings.HasPrefix(name, "/") {
			log.Printf("content: import skips unsafe path %q", f.Name)
			continue
		}
		segs := strings.Split(name, "/")
		if len(segs) == 1 && segs[0] == RegistryFile {
			s.mergeRegistryZip(f)
			continue
		}
		if len(segs) != 2 || !strings.HasSuffix(segs[1], ".json") {
			continue
		}
		if !isContentDir(segs[0]) {
			log.Printf("content: import skips unknown folder %q", f.Name)
			continue
		}
		data, err := readZip(f)
		if err != nil {
			log.Printf("content: import read %s: %v", f.Name, err)
			continue
		}
		id, err := s.writeTyped(segs[0], data)
		if err != nil {
			log.Printf("content: import skips %s: %v", f.Name, err)
			continue
		}
		log.Printf("content: imported %s (%s)", f.Name, id)
	}
	return nil
}

func isContentDir(dir string) bool {
	switch dir {
	case DirSpecies, DirWater, DirPlants, DirPatterns, DirCorals:
		return true
	}
	return false
}

func readZip(f *zip.File) ([]byte, error) {
	rc, err := f.Open()
	if err != nil {
		return nil, err
	}
	defer rc.Close()
	return io.ReadAll(rc)
}

// mergeRegistryZip folds a pack's registry.json into the local registry.
func (s *Store) mergeRegistryZip(f *zip.File) {
	data, err := readZip(f)
	if err != nil {
		log.Printf("content: import registry read: %v", err)
		return
	}
	entries, err := decodeRegistry(data)
	if err != nil {
		log.Printf("content: import registry skipped: %v", err)
		return
	}
	added := s.reg.Merge(entries)
	log.Printf("content: registry merged from pack: %d new entries", added)
}
