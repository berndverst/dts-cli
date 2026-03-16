//go:build integration

package integration

import (
	"strings"
	"testing"
)

func TestWorkerList(t *testing.T) {
	stdout, _ := runDTS(t, "exec", "work", "list")
	if stdout == "" {
		t.Fatal("Expected non-empty response from workers list")
	}
	stdout = strings.TrimSpace(stdout)
	if stdout[0] == '[' {
		parseJSONArray(t, stdout)
	} else {
		parseJSON(t, stdout)
	}
}
