//go:build integration

package integration

import "testing"

func TestEntityList(t *testing.T) {
	stdout, _ := runDTS(t, "exec", "ent", "list")
	// Should return valid JSON with entities array
	m := parseJSON(t, stdout)
	if _, ok := m["entities"]; !ok {
		t.Fatal("Expected 'entities' key in response")
	}
}
