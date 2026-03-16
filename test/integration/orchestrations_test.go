//go:build integration

package integration

import (
	"encoding/json"
	"fmt"
	"testing"
	"time"
)

// TestOrchestrationLifecycle exercises the full orchestration command set in order.
// Subtests are sequential since each depends on state from previous steps.
func TestOrchestrationLifecycle(t *testing.T) {
	var instanceID string

	t.Run("List_Empty", func(t *testing.T) {
		stdout, _ := runDTS(t, "exec", "orch", "list")
		m := parseJSON(t, stdout)
		if _, ok := m["orchestrations"]; !ok {
			t.Fatal("Expected 'orchestrations' key in response")
		}
	})

	t.Run("Create", func(t *testing.T) {
		stdout, _ := runDTS(t, "exec", "orch", "create",
			"--name", "IntegrationTestOrch",
			"--input", `{"testKey":"testValue"}`,
		)
		m := parseJSON(t, stdout)
		id, ok := m["instanceId"].(string)
		if !ok || id == "" {
			t.Fatalf("Expected non-empty instanceId, got: %s", stdout)
		}
		instanceID = id
		t.Logf("Created orchestration: %s", instanceID)
	})

	if instanceID == "" {
		t.Fatal("Cannot continue without instance ID from Create step")
	}

	time.Sleep(500 * time.Millisecond)

	t.Run("List_NotEmpty", func(t *testing.T) {
		stdout, _ := runDTS(t, "exec", "orch", "list")
		m := parseJSON(t, stdout)
		orchs, ok := m["orchestrations"].([]interface{})
		if !ok {
			t.Fatal("Expected orchestrations array")
		}
		if len(orchs) == 0 {
			t.Fatal("Expected at least one orchestration after create")
		}
	})

	t.Run("Get", func(t *testing.T) {
		stdout, _ := runDTS(t, "exec", "orch", "get", instanceID)
		m := parseJSON(t, stdout)
		if m["instanceId"] != instanceID {
			t.Fatalf("Expected instanceId=%s, got %v", instanceID, m["instanceId"])
		}
	})

	t.Run("Payloads", func(t *testing.T) {
		stdout, _ := runDTS(t, "exec", "orch", "payloads", instanceID)
		parseJSON(t, stdout)
	})

	t.Run("History", func(t *testing.T) {
		stdout, _ := runDTS(t, "exec", "orch", "history", instanceID)
		var arr []json.RawMessage
		if err := json.Unmarshal([]byte(stdout), &arr); err != nil {
			// Some emulator versions return an object wrapper
			parseJSON(t, stdout)
		}
	})

	t.Run("Suspend", func(t *testing.T) {
		stdout, _ := runDTS(t, "exec", "orch", "suspend", instanceID, "--reason", "integration test")
		m := parseJSON(t, stdout)
		if m["status"] != "ok" {
			t.Fatalf("Expected status=ok, got %v", m["status"])
		}
	})

	time.Sleep(500 * time.Millisecond)

	t.Run("Resume", func(t *testing.T) {
		stdout, _ := runDTS(t, "exec", "orch", "resume", instanceID, "--reason", "integration test")
		m := parseJSON(t, stdout)
		if m["status"] != "ok" {
			t.Fatalf("Expected status=ok, got %v", m["status"])
		}
	})

	time.Sleep(500 * time.Millisecond)

	t.Run("Terminate", func(t *testing.T) {
		stdout, _ := runDTS(t, "exec", "orch", "terminate", instanceID, "--reason", "integration test")
		m := parseJSON(t, stdout)
		if m["status"] != "ok" {
			t.Fatalf("Expected status=ok, got %v", m["status"])
		}
	})

	time.Sleep(500 * time.Millisecond)

	t.Run("RaiseEvent", func(t *testing.T) {
		// Create a fresh orchestration for the event test
		stdout, _ := runDTS(t, "exec", "orch", "create",
			"--name", "IntegrationTestOrch",
			"--input", `{"testKey":"eventTest"}`,
		)
		m := parseJSON(t, stdout)
		eventInstanceID := m["instanceId"].(string)

		time.Sleep(500 * time.Millisecond)

		stdout, _ = runDTS(t, "exec", "orch", "raise-event", eventInstanceID,
			"--event-name", "TestEvent",
			"--data", `{"approved":true}`,
		)
		m = parseJSON(t, stdout)
		if m["status"] != "ok" {
			t.Fatalf("Expected status=ok, got %v", m["status"])
		}

		// Clean up
		runDTSRaw("exec", "orch", "purge", eventInstanceID)
	})

	t.Run("Restart", func(t *testing.T) {
		stdout, _ := runDTS(t, "exec", "orch", "restart", instanceID)
		m := parseJSON(t, stdout)
		// The emulator returns {"id", "result", "status"} — verify it parsed
		if m["status"] == nil {
			t.Fatalf("Expected status field in restart response, got: %s", stdout)
		}
		t.Logf("Restart response: %s", stdout)
	})

	t.Run("ForceTerminate", func(t *testing.T) {
		// Create a fresh orchestration
		stdout, _ := runDTS(t, "exec", "orch", "create",
			"--name", "IntegrationTestOrch",
		)
		m := parseJSON(t, stdout)
		ftID := m["instanceId"].(string)

		time.Sleep(500 * time.Millisecond)

		stdout, _ = runDTS(t, "exec", "orch", "force-terminate",
			"--ids", ftID,
			"--reason", "integration test",
		)
		m = parseJSON(t, stdout)
		if m["status"] != "ok" {
			t.Fatalf("Expected status=ok, got %v", m["status"])
		}

		time.Sleep(500 * time.Millisecond)

		// Clean up
		runDTSRaw("exec", "orch", "purge", ftID)
	})

	t.Run("Purge", func(t *testing.T) {
		stdout, _ := runDTS(t, "exec", "orch", "purge", instanceID)
		m := parseJSON(t, stdout)
		if m["status"] != "ok" {
			t.Fatalf("Expected status=ok, got %v", m["status"])
		}
	})

	t.Run("Create_WithInstanceID", func(t *testing.T) {
		customID := fmt.Sprintf("test-%d", time.Now().UnixNano())
		stdout, _ := runDTS(t, "exec", "orch", "create",
			"--name", "IntegrationTestOrch",
			"--instance-id", customID,
			"--input", `{"key":"value"}`,
		)
		m := parseJSON(t, stdout)
		id, ok := m["instanceId"].(string)
		if !ok || id == "" {
			t.Fatalf("Expected instanceId, got: %s", stdout)
		}
		t.Logf("Created with custom ID: %s", id)

		// Clean up
		time.Sleep(500 * time.Millisecond)
		runDTSRaw("exec", "orch", "purge", id)
	})
}
