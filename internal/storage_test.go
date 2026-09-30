package clicrawler

import (
	"bytes"
	"encoding/json/jsontext"
	"encoding/json/v2"
	"log"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestStorage_SaveRoundTrip(t *testing.T) {
	outPath := filepath.Join(t.TempDir(), "result.json")
	var logBuf bytes.Buffer
	storage := &JSONStorage{filepath: outPath, logger: *log.New(&logBuf, "[STORAGE] ", 0)}

	want := []ResourseNode{{Resourse: "https://a.com", Title: "Page A", Links: []ResourseNode{
		{Resourse: "https://a.com/b", Title: "Page B"},
	}}}

	data, err := storage.Save(want)
	if err != nil {
		t.Fatalf("Save failed: %v", err)
	}
	var got []ResourseNode
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("saved data is not valid JSON: %v", err)
	}
	if len(got) != 1 || got[0].Resourse != "https://a.com" || got[0].Title != "Page A" {
		t.Fatalf("root mismatch: %+v", got)
	}
	if len(got[0].Links) != 1 || got[0].Links[0].Resourse != "https://a.com/b" || got[0].Links[0].Title != "Page B" {
		t.Errorf("child mismatch: %+v", got[0].Links)
	}
	raw, err := os.ReadFile(outPath)
	if err != nil {
		t.Fatalf("result file was not written: %v", err)
	}
	if string(raw) != string(data) {
		t.Errorf("file content differs from returned data:\nfile %s\nret  %s", raw, data)
	}
}

func TestStorage_OutputSurvivesInvalidUTF8(t *testing.T) {
	broken := []ResourseNode{{Resourse: "example.com", Title: "Google \xcc\xe0\xeb"}}
	
	if _, err := json.Marshal(broken); err == nil {
		t.Error("default Marshal must reject invalid UTF-8 — this is why Crawle passes jsontext.AllowInvalidUTF8")
	}

	data, err := json.Marshal(broken, jsontext.AllowInvalidUTF8(true))
	if err != nil {
		t.Fatalf("Marshal with AllowInvalidUTF8 failed: %v", err)
	}
	if err := json.Unmarshal(data, new(any)); err != nil {
		t.Errorf("output is not valid JSON: %v (%s)", err, data)
	}
	if !strings.ContainsRune(string(data), '�') {
		t.Errorf("invalid bytes must be mangled to U+FFFD, got %s", data)
	}
}
