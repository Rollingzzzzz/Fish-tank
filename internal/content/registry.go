// G3.6: append-only fingerprint registry (content/registry.json), per C6.
// Entries are {hash, kind, id, at}; deduped by hash; capped at
// contract.RegistryCap with the oldest entry dropped on overflow.
package content

import (
	"encoding/json"
	"log"
	"os"
	"sync"
	"time"

	"github.com/Rollingzzzzz/Fish-tank/internal/contract"
)

// RegistryEntry is one accepted-artifact record in registry.json.
type RegistryEntry struct {
	Hash string `json:"hash"`
	Kind string `json:"kind"`
	ID   string `json:"id"`
	At   string `json:"at"`
}

// Registry is the goroutine-safe in-memory view of registry.json.
type Registry struct {
	mu      sync.Mutex
	path    string
	entries []RegistryEntry
}

// LoadRegistry reads path; a missing or corrupt file yields an empty registry
// with a log line, never an error to the caller (D4). The error return exists
// only for hard I/O failures creating the store.
func LoadRegistry(path string) (*Registry, error) {
	r := &Registry{path: path}
	b, err := os.ReadFile(path)
	if err != nil {
		if !os.IsNotExist(err) {
			log.Printf("content: read registry %s: %v", path, err)
		}
		return r, nil
	}
	var entries []RegistryEntry
	if err := json.Unmarshal(b, &entries); err != nil {
		log.Printf("content: corrupt registry %s ignored: %v", path, err)
		return r, nil
	}
	if len(entries) > contract.RegistryCap {
		entries = entries[len(entries)-contract.RegistryCap:]
	}
	r.entries = entries
	return r, nil
}

// Append records hash (deduped by hash, capped) and persists to disk.
func (r *Registry) Append(kind, id, hash string) {
	r.appendEntry(RegistryEntry{Hash: hash, Kind: kind, ID: id, At: nowRFC3339()})
	r.Save()
}

// appendEntry adds one entry without saving (cap + dedupe enforced).
func (r *Registry) appendEntry(e RegistryEntry) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.hasLocked(e.Hash) {
		return
	}
	r.entries = append(r.entries, e)
	if len(r.entries) > contract.RegistryCap {
		drop := len(r.entries) - contract.RegistryCap
		r.entries = r.entries[drop:]
	}
}

// Merge folds entries from an imported pack, deduped by hash; returns the
// number of newly added entries and persists once.
func (r *Registry) Merge(entries []RegistryEntry) int {
	r.mu.Lock()
	added := 0
	for _, e := range entries {
		if r.hasLocked(e.Hash) {
			continue
		}
		r.entries = append(r.entries, e)
		added++
	}
	if len(r.entries) > contract.RegistryCap {
		r.entries = r.entries[len(r.entries)-contract.RegistryCap:]
	}
	r.mu.Unlock()
	if added > 0 {
		r.Save()
	}
	return added
}

// Has reports whether hash is registered.
func (r *Registry) Has(hash string) bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.hasLocked(hash)
}

func (r *Registry) hasLocked(hash string) bool {
	for _, e := range r.entries {
		if e.Hash == hash {
			return true
		}
	}
	return false
}

// Recent returns up to n hashes, newest last.
func (r *Registry) Recent(n int) []string {
	r.mu.Lock()
	defer r.mu.Unlock()
	if n <= 0 || len(r.entries) == 0 {
		return nil
	}
	if n > len(r.entries) {
		n = len(r.entries)
	}
	out := make([]string, 0, n)
	for _, e := range r.entries[len(r.entries)-n:] {
		out = append(out, e.Hash)
	}
	return out
}

// Len returns the number of registered entries.
func (r *Registry) Len() int {
	r.mu.Lock()
	defer r.mu.Unlock()
	return len(r.entries)
}

// Entries returns a copy of all entries (oldest first).
func (r *Registry) Entries() []RegistryEntry {
	r.mu.Lock()
	defer r.mu.Unlock()
	return append([]RegistryEntry(nil), r.entries...)
}

// Save persists the registry atomically; failures are logged, never fatal (D4).
func (r *Registry) Save() {
	r.mu.Lock()
	data, err := json.MarshalIndent(r.entries, "", "  ")
	r.mu.Unlock()
	if err != nil {
		log.Printf("content: marshal registry: %v", err)
		return
	}
	data = append(data, '\n')
	if err := atomicWrite(r.path, data); err != nil {
		log.Printf("content: save registry: %v", err)
	}
}

func nowRFC3339() string { return time.Now().UTC().Format(time.RFC3339) }

// decodeRegistry parses raw registry.json bytes (shared with pack import).
func decodeRegistry(data []byte) ([]RegistryEntry, error) {
	var entries []RegistryEntry
	if err := json.Unmarshal(data, &entries); err != nil {
		return nil, err
	}
	return entries, nil
}
