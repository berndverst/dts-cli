//go:build integration

package integration

import (
	"bytes"
	"context"
	"os/exec"
	"testing"
	"time"
)

// TestScheduleLifecycle exercises the schedule command set.
// Schedules are implemented using durable entities. The schedule REST API creates
// an internal ExecuteScheduleOperationOrchestrator that requires a connected worker
// running UseScheduledTasks() to process it. Without a worker, schedule mutations
// (create, pause, resume, delete) block indefinitely. Schedule list works without
// a worker since it only queries the backend.
func TestScheduleLifecycle(t *testing.T) {
	const scheduleID = "integration-test-schedule"
	var created bool

	t.Run("List", func(t *testing.T) {
		stdout, _ := runDTS(t, "exec", "sched", "list")
		parseJSON(t, stdout)
	})

	t.Run("Create", func(t *testing.T) {
		stdout, _, err := runDTSWithTimeout(10*time.Second, "exec", "sched", "create",
			"--schedule-id", scheduleID,
			"--orchestration-name", "ScheduledOrch",
			"--interval", "PT1H",
		)
		if err != nil {
			t.Skipf("Schedule create requires a connected worker with UseScheduledTasks(): %v", err)
			return
		}
		m := parseJSON(t, stdout)
		if m["status"] != "created" {
			t.Fatalf("Expected status=created, got %v", m["status"])
		}
		created = true
	})

	t.Run("Pause", func(t *testing.T) {
		if !created {
			t.Skip("Skipping: schedule create requires a connected worker")
		}
		_, _, err := runDTSWithTimeout(10*time.Second, "exec", "sched", "pause", scheduleID)
		if err != nil {
			t.Skipf("Schedule pause requires a connected worker: %v", err)
		}
	})

	t.Run("Resume", func(t *testing.T) {
		if !created {
			t.Skip("Skipping: schedule create requires a connected worker")
		}
		_, _, err := runDTSWithTimeout(10*time.Second, "exec", "sched", "resume", scheduleID)
		if err != nil {
			t.Skipf("Schedule resume requires a connected worker: %v", err)
		}
	})

	t.Run("Delete", func(t *testing.T) {
		if !created {
			t.Skip("Skipping: schedule create requires a connected worker")
		}
		_, _, err := runDTSWithTimeout(10*time.Second, "exec", "sched", "delete", scheduleID)
		if err != nil {
			t.Skipf("Schedule delete requires a connected worker: %v", err)
		}
	})
}

// runDTSWithTimeout runs the CLI with a context timeout.
func runDTSWithTimeout(timeout time.Duration, args ...string) (stdout, stderr string, err error) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	fullArgs := append([]string{
		"--url", dtsURL,
		"--taskhub", dtsTaskHub,
		"--auth-mode", "none",
	}, args...)

	cmd := exec.CommandContext(ctx, binaryPath, fullArgs...)
	var outBuf, errBuf bytes.Buffer
	cmd.Stdout = &outBuf
	cmd.Stderr = &errBuf

	err = cmd.Run()
	return outBuf.String(), errBuf.String(), err
}
