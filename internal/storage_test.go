package clicrawler

import (
	"encoding/json/jsontext"
	"encoding/json/v2"
	"strings"
	"testing"
)

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
