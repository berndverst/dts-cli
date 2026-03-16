//go:build integration

package integration

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"
)

// binaryPath is the path to the compiled dts binary.
var binaryPath string

// dtsURL is the DTS emulator endpoint (port 8081 is the HTTP/1.1 REST API).
var dtsURL = envOrDefault("DTS_URL", "http://localhost:8081")

// dtsTaskHub is the task hub name.
var dtsTaskHub = envOrDefault("DTS_TASKHUB", "default")

func TestMain(m *testing.M) {
	// Build the binary
	tmpDir, err := os.MkdirTemp("", "dts-integration-*")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to create temp dir: %v\n", err)
		os.Exit(1)
	}
	defer os.RemoveAll(tmpDir)

	binaryPath = filepath.Join(tmpDir, "dts")
	buildCmd := exec.Command("go", "build", "-o", binaryPath, ".")
	buildCmd.Dir = filepath.Join(repoRoot())
	buildCmd.Stdout = os.Stdout
	buildCmd.Stderr = os.Stderr
	if err := buildCmd.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to build binary: %v\n", err)
		os.Exit(1)
	}

	// Wait for emulator to be ready
	if err := waitForEmulator(60 * time.Second); err != nil {
		fmt.Fprintf(os.Stderr, "Emulator not ready: %v\n", err)
		os.Exit(1)
	}

	os.Exit(m.Run())
}

// runDTS executes the dts binary with the given arguments, prepending common connection flags.
func runDTS(t *testing.T, args ...string) (stdout, stderr string) {
	t.Helper()
	fullArgs := append([]string{
		"--url", dtsURL,
		"--taskhub", dtsTaskHub,
		"--auth-mode", "none",
	}, args...)

	cmd := exec.Command(binaryPath, fullArgs...)
	var outBuf, errBuf bytes.Buffer
	cmd.Stdout = &outBuf
	cmd.Stderr = &errBuf

	err := cmd.Run()
	if err != nil {
		t.Fatalf("dts %v failed: %v\nstdout: %s\nstderr: %s", args, err, outBuf.String(), errBuf.String())
	}
	return outBuf.String(), errBuf.String()
}

// runDTSRaw executes the dts binary and returns output plus any error (does not call t.Fatal).
func runDTSRaw(args ...string) (stdout, stderr string, err error) {
	fullArgs := append([]string{
		"--url", dtsURL,
		"--taskhub", dtsTaskHub,
		"--auth-mode", "none",
	}, args...)

	cmd := exec.Command(binaryPath, fullArgs...)
	var outBuf, errBuf bytes.Buffer
	cmd.Stdout = &outBuf
	cmd.Stderr = &errBuf

	err = cmd.Run()
	return outBuf.String(), errBuf.String(), err
}

// parseJSON unmarshals a JSON string into a map.
func parseJSON(t *testing.T, s string) map[string]interface{} {
	t.Helper()
	var m map[string]interface{}
	if err := json.Unmarshal([]byte(s), &m); err != nil {
		t.Fatalf("Failed to parse JSON: %v\nRaw: %s", err, s)
	}
	return m
}

// parseJSONArray unmarshals a JSON array string.
func parseJSONArray(t *testing.T, s string) []interface{} {
	t.Helper()
	var arr []interface{}
	if err := json.Unmarshal([]byte(s), &arr); err != nil {
		t.Fatalf("Failed to parse JSON array: %v\nRaw: %s", err, s)
	}
	return arr
}

// waitForEmulator polls the emulator ping endpoint until it responds or timeout.
func waitForEmulator(timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	client := &http.Client{Timeout: 2 * time.Second}
	for time.Now().Before(deadline) {
		req, _ := http.NewRequest(http.MethodGet, dtsURL+"/v1/taskhubs/ping", nil)
		req.Header.Set("x-taskhub", dtsTaskHub)
		resp, err := client.Do(req)
		if err == nil {
			resp.Body.Close()
			if resp.StatusCode >= 200 && resp.StatusCode < 300 {
				return nil
			}
		}
		time.Sleep(1 * time.Second)
	}
	return fmt.Errorf("emulator at %s not ready after %v", dtsURL, timeout)
}

// repoRoot returns the repository root directory.
func repoRoot() string {
	// Walk up from the test file location to find go.mod
	dir, _ := os.Getwd()
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	// Fallback: assume we're two levels deep from repo root
	wd, _ := os.Getwd()
	return filepath.Join(wd, "..", "..")
}

func envOrDefault(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
