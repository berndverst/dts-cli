//go:build integration

package integration

import "testing"

func TestAgentList(t *testing.T) {
	// Agents may not be fully supported by the emulator.
	stdout, _, err := runDTSRaw("exec", "ag", "list")
	if err != nil {
		t.Skipf("Agent list not supported by emulator: %v", err)
		return
	}

	// Agent list returns an object with an agents array
	m := parseJSON(t, stdout)
	if _, ok := m["agents"]; !ok {
		t.Fatal("Expected 'agents' key in agent list response")
	}
}
