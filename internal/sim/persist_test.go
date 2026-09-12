// G2.4/G2.5: snapshot round-trip and 3-tier crash recovery tests.
package sim

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestSnapshotRestoreRoundTrip(t *testing.T) {
	w := testWorld(t, testSpecies(0.5), 4)
	for i := 0; i < 30; i++ {
		w.Update(1/60.0, Input{})
	}
	before := w.Snapshot()
	// mutate
	for i := 0; i < 30; i++ {
		w.Update(1/60.0, Input{})
	}
	if err := w.Restore(before); err != nil {
		t.Fatal(err)
	}
	after := w.Snapshot()
	if len(after.Fish) != len(before.Fish) {
		t.Fatalf("census mismatch: %d vs %d", len(after.Fish), len(before.Fish))
	}
	for i := range after.Fish {
		if mathAbs(after.Fish[i].Pos.X-before.Fish[i].Pos.X) > 1e-6 ||
			mathAbs(after.Fish[i].AgeDays-before.Fish[i].AgeDays) > 1e-9 ||
			mathAbs(after.Fish[i].Satiety-before.Fish[i].Satiety) > 1e-6 {
			t.Fatalf("fish %d state not restored", i)
		}
	}
}

func mathAbs(x float64) float64 {
	if x < 0 {
		return -x
	}
	return x
}

func TestTierRecoveryFallsBackToDaily(t *testing.T) {
	dir := t.TempDir()
	w := testWorld(t, testSpecies(0), 3)
	base := time.Now()
	p := NewPersist(dir, base)
	p.Tick(w, base)                          // seeds daily + weekly + minute
	if err := p.SnapshotNow(w); err != nil { // fresh minute
		t.Fatal(err)
	}
	// verify minute loads
	if s, err := LoadLatest(dir); err != nil || len(s.Fish) != 3 {
		t.Fatalf("minute tier not loaded: %v", err)
	}
	// corrupt the minute file, write a good daily
	minute := filepath.Join(dir, fileMinute)
	if err := os.WriteFile(minute, []byte("{corrupt"), 0o644); err != nil {
		t.Fatal(err)
	}
	p2 := NewPersist(dir, base.AddDate(0, 0, 1))
	if err := p2.Tick(w, base.AddDate(0, 0, 1)); err != nil { // new day → daily
		t.Fatal(err)
	}
	s, err := LoadLatest(dir)
	if err != nil || len(s.Fish) != 3 {
		t.Fatalf("daily fallback failed: %v %d", err, len(s.Fish))
	}
}

func TestHighSchemaVersionSkipped(t *testing.T) {
	dir := t.TempDir()
	w := testWorld(t, testSpecies(0), 2)
	base := time.Now()
	p := NewPersist(dir, base)
	p.Tick(w, base) // writes all three tiers
	// poison every file with a future schema version
	for _, name := range []string{fileMinute, fileDaily, fileWeekly} {
		path := filepath.Join(dir, name)
		if err := os.WriteFile(path, []byte(`{"schemaVersion":99,"fish":[]}`), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := LoadLatest(dir); err == nil {
		t.Fatal("future schema files must be skipped, fresh world expected")
	}
}

func TestDailySnapshotFiresOncePerDay(t *testing.T) {
	dir := t.TempDir()
	w := testWorld(t, testSpecies(0), 2)
	// fixed synthetic morning time — never crosses midnight mid-test
	base := time.Date(2026, 9, 15, 10, 0, 0, 0, time.Local)
	p := NewPersist(dir, base)
	daily := filepath.Join(dir, fileDaily)
	p.Tick(w, base)
	st1 := statMod(daily)
	// same day, much later — daily must not rewrite
	p.Tick(w, base.Add(6*time.Hour))
	st2 := statMod(daily)
	if !st1.Equal(st2) {
		t.Fatal("daily snapshot rewritten within the same day")
	}
	// next day — rewrites once
	p.Tick(w, base.AddDate(0, 0, 1))
	if statMod(daily).Equal(st2) {
		t.Fatal("daily snapshot did not refresh on day rollover")
	}
}

func statMod(path string) time.Time {
	fi, err := os.Stat(path)
	if err != nil {
		return time.Time{}
	}
	return fi.ModTime()
}
