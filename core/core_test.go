package core

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestFormatJSON(t *testing.T) {
	raw := `{"name":"octonote","active":true}`
	prettified, err := FormatJSON(raw, false)
	if err != nil {
		t.Fatalf("FormatJSON failed: %v", err)
	}
	if len(prettified) <= len(raw) {
		t.Fatalf("expected prettified JSON to be longer than raw JSON")
	}

	minified, err := FormatJSON(prettified, true)
	if err != nil {
		t.Fatalf("FormatJSON minify failed: %v", err)
	}

	var originalMap, minifiedMap map[string]interface{}
	if err := json.Unmarshal([]byte(raw), &originalMap); err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal([]byte(minified), &minifiedMap); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(originalMap, minifiedMap) {
		t.Fatalf("expected equivalent JSON, got %v vs %v", originalMap, minifiedMap)
	}
}

func TestTransformCase(t *testing.T) {
	text := "hello_world_note"
	if got := TransformCase(text, "upper"); got != "HELLO_WORLD_NOTE" {
		t.Errorf("upper: got %q", got)
	}
	if got := TransformCase(text, "camel"); got != "helloWorldNote" {
		t.Errorf("camel: got %q", got)
	}
	if got := TransformCase("helloWorldNote", "snake"); got != "hello_world_note" {
		t.Errorf("snake: got %q", got)
	}
	if got := TransformCase("helloWorldNote", "kebab"); got != "hello-world-note" {
		t.Errorf("kebab: got %q", got)
	}
}

func TestCalculateMetrics(t *testing.T) {
	text := "octoNote is a lightweight scratchpad.\nWith multiple lines."
	metrics := CalculateMetrics(text)
	if metrics["words"].(int) != 8 {
		t.Errorf("expected 8 words, got %v", metrics["words"])
	}
	if metrics["lines"].(int) != 2 {
		t.Errorf("expected 2 lines, got %v", metrics["lines"])
	}
}

func TestStorageLoadSave(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "octonote-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	s := &Storage{
		dir:  tmpDir,
		file: filepath.Join(tmpDir, "state.json"),
		ch:   make(chan State, 64),
		done: make(chan struct{}),
	}
	go s.syncWriter()

	st, err := s.Load()
	if err != nil {
		t.Fatalf("initial Load error: %v", err)
	}
	if len(st.Tabs) != 1 || st.Tabs[0].Title != "scratch" {
		t.Fatalf("unexpected default state: %+v", st)
	}

	// Add tab and save
	st.Tabs = append(st.Tabs, NewTab("ideas"))
	s.Save(st)

	// Close cleanly drains writer
	s.Close()

	// Reopen storage and verify state
	s2 := &Storage{
		dir:  tmpDir,
		file: filepath.Join(tmpDir, "state.json"),
		ch:   make(chan State, 64),
		done: make(chan struct{}),
	}
	go s2.syncWriter()
	defer s2.Close()

	loaded, err := s2.Load()
	if err != nil {
		t.Fatalf("Load error after save: %v", err)
	}
	if len(loaded.Tabs) != 2 {
		t.Fatalf("expected 2 tabs after save, got %d", len(loaded.Tabs))
	}
	if loaded.Tabs[1].Title != "ideas" {
		t.Fatalf("expected tab title 'ideas', got %q", loaded.Tabs[1].Title)
	}
}

func TestSanitiseTitle(t *testing.T) {
	dirty := "my\x00tab\r\nname"
	clean := SanitiseTitle(dirty)
	if clean != "mytabname" {
		t.Errorf("expected 'mytabname', got %q", clean)
	}
}
