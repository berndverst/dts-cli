//go:build integration

package integration

import "testing"

func TestPing(t *testing.T) {
	stdout, _ := runDTS(t, "exec", "ping")
	m := parseJSON(t, stdout)
	if m["status"] != "ok" {
		t.Fatalf("Expected status=ok, got %v", m["status"])
	}
}
