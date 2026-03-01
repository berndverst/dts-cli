package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
)

// QueryOrchestrations queries orchestrations with filtering, sorting, and pagination.
func (c *Client) QueryOrchestrations(ctx context.Context, req *QueryOrchestrationsRequest) (*OrchestrationsResult, error) {
	var result OrchestrationsResult
	if err := c.doJSON(ctx, http.MethodPost, "/v1/taskhubs/orchestrations/query", req, &result); err != nil {
		return nil, fmt.Errorf("querying orchestrations: %w", err)
	}
	return &result, nil
}

// GetOrchestration gets a single orchestration by instance ID.
func (c *Client) GetOrchestration(ctx context.Context, instanceID string) (*Orchestration, error) {
	var result Orchestration
	if err := c.doJSON(ctx, http.MethodGet, "/v1/taskhubs/orchestrations/"+instanceID, nil, &result); err != nil {
		return nil, fmt.Errorf("getting orchestration %s: %w", instanceID, err)
	}
	return &result, nil
}

// GetOrchestrationPayloads gets input/output/failure details for an orchestration.
func (c *Client) GetOrchestrationPayloads(ctx context.Context, instanceID string) (*OrchestrationPayloads, error) {
	var result OrchestrationPayloads
	path := fmt.Sprintf("/v1/taskhubs/orchestrations/%s/payloads", instanceID)
	if err := c.doJSON(ctx, http.MethodGet, path, nil, &result); err != nil {
		return nil, fmt.Errorf("getting orchestration payloads %s: %w", instanceID, err)
	}
	return &result, nil
}

// GetOrchestrationHistory gets the execution history for an orchestration.
func (c *Client) GetOrchestrationHistory(ctx context.Context, instanceID, executionID string) ([]HistoryEvent, error) {
	path := fmt.Sprintf("/v1/taskhubs/orchestrations/%s/executions/%s/history", instanceID, executionID)

	resp, err := c.doRequest(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, fmt.Errorf("getting orchestration history: %w", err)
	}
	defer resp.Body.Close()

	if err := checkResponse(resp); err != nil {
		return nil, err
	}

	// History is returned as a JSON array of protobuf-json events
	var events []HistoryEvent
	if err := decodeJSON(resp, &events); err != nil {
		return nil, fmt.Errorf("decoding history: %w", err)
	}
	return events, nil
}

// CreateOrchestration creates a new orchestration instance.
func (c *Client) CreateOrchestration(ctx context.Context, req *CreateOrchestrationRequest) (string, error) {
	resp, err := c.doRequest(ctx, http.MethodPost, "/v1/taskhubs/orchestrations", req)
	if err != nil {
		return "", fmt.Errorf("creating orchestration: %w", err)
	}
	defer resp.Body.Close()

	if err := checkResponse(resp); err != nil {
		return "", err
	}

	// Instance ID is in the Location header or response body
	loc := resp.Header.Get("Location")
	if loc != "" {
		// Location: orchestrations/{instanceId}
		parts := splitLast(loc, "/")
		if len(parts) == 2 {
			return parts[1], nil
		}
	}
	// Fallback: read body
	body, _, _ := readBody(resp)
	return body, nil
}

// RestartOrchestration restarts an orchestration.
func (c *Client) RestartOrchestration(ctx context.Context, instanceID string, newInstanceID bool) (string, error) {
	req := &RestartOrchestrationRequest{RestartWithNewInstanceID: newInstanceID}
	body, _, err := c.doRaw(ctx, http.MethodPost, "/v1/taskhubs/orchestrations/"+instanceID+"/restart", req)
	if err != nil {
		return "", fmt.Errorf("restarting orchestration %s: %w", instanceID, err)
	}
	return body, nil
}

// PurgeOrchestration deletes a single orchestration.
func (c *Client) PurgeOrchestration(ctx context.Context, instanceID string) error {
	if err := c.doNoContent(ctx, http.MethodDelete, "/v1/taskhubs/orchestrations/"+instanceID, nil); err != nil {
		return fmt.Errorf("purging orchestration %s: %w", instanceID, err)
	}
	return nil
}

// PurgeOrchestrations deletes multiple orchestrations.
func (c *Client) PurgeOrchestrations(ctx context.Context, instanceIDs []string) error {
	if err := c.doNoContent(ctx, http.MethodDelete, "/v1/taskhubs/orchestrations/purge", instanceIDs); err != nil {
		return fmt.Errorf("purging orchestrations: %w", err)
	}
	return nil
}

// SuspendOrchestration suspends a running orchestration.
func (c *Client) SuspendOrchestration(ctx context.Context, instanceID, reason string) error {
	event := map[string]interface{}{
		"executionSuspended": map[string]interface{}{
			"reason": reason,
		},
	}
	return c.addHistoryEvent(ctx, instanceID, event)
}

// ResumeOrchestration resumes a suspended orchestration.
func (c *Client) ResumeOrchestration(ctx context.Context, instanceID, reason string) error {
	event := map[string]interface{}{
		"executionResumed": map[string]interface{}{
			"reason": reason,
		},
	}
	return c.addHistoryEvent(ctx, instanceID, event)
}

// TerminateOrchestration terminates an orchestration.
func (c *Client) TerminateOrchestration(ctx context.Context, instanceID, reason string) error {
	event := map[string]interface{}{
		"executionTerminated": map[string]interface{}{
			"reason": reason,
		},
	}
	return c.addHistoryEvent(ctx, instanceID, event)
}

// RewindOrchestration rewinds a failed orchestration.
func (c *Client) RewindOrchestration(ctx context.Context, instanceID, reason string) error {
	event := map[string]interface{}{
		"executionRewound": map[string]interface{}{
			"reason": reason,
		},
	}
	return c.addHistoryEvent(ctx, instanceID, event)
}

// RaiseEvent sends a named event to an orchestration.
func (c *Client) RaiseEvent(ctx context.Context, instanceID, eventName, eventData string) error {
	event := map[string]interface{}{
		"eventRaised": map[string]interface{}{
			"name":  eventName,
			"input": eventData,
		},
	}
	return c.addHistoryEvent(ctx, instanceID, event)
}

// ForceTerminate force-terminates multiple orchestrations.
func (c *Client) ForceTerminate(ctx context.Context, instanceIDs []string, reason string) ([]string, error) {
	req := &ForceTerminateRequest{
		InstanceIDs: instanceIDs,
		Reason:      reason,
	}
	var unsuccessful []string
	if err := c.doJSON(ctx, http.MethodPost, "/v1/taskhubs/orchestrations/forceterminate", req, &unsuccessful); err != nil {
		return nil, fmt.Errorf("force terminating: %w", err)
	}
	return unsuccessful, nil
}

// BatchSuspend suspends multiple orchestrations.
func (c *Client) BatchSuspend(ctx context.Context, instanceIDs []string, reason string) ([]string, error) {
	return c.batchAddEvents(ctx, instanceIDs, map[string]interface{}{
		"executionSuspended": map[string]interface{}{"reason": reason},
	})
}

// BatchResume resumes multiple orchestrations.
func (c *Client) BatchResume(ctx context.Context, instanceIDs []string, reason string) ([]string, error) {
	return c.batchAddEvents(ctx, instanceIDs, map[string]interface{}{
		"executionResumed": map[string]interface{}{"reason": reason},
	})
}

// BatchTerminate terminates multiple orchestrations.
func (c *Client) BatchTerminate(ctx context.Context, instanceIDs []string, reason string) ([]string, error) {
	return c.batchAddEvents(ctx, instanceIDs, map[string]interface{}{
		"executionTerminated": map[string]interface{}{"reason": reason},
	})
}

func (c *Client) addHistoryEvent(ctx context.Context, instanceID string, event interface{}) error {
	path := fmt.Sprintf("/v1/taskhubs/orchestrations/%s/history", instanceID)
	resp, err := c.doRequest(ctx, http.MethodPost, path, event)
	if err != nil {
		return fmt.Errorf("adding history event: %w", err)
	}
	defer resp.Body.Close()
	return checkResponse(resp)
}

func (c *Client) batchAddEvents(ctx context.Context, instanceIDs []string, event interface{}) ([]string, error) {
	events := make(map[string]interface{})
	for _, id := range instanceIDs {
		events[id] = event
	}
	req := &BatchHistoryEventsRequest{Events: events}
	var unsuccessful []string
	if err := c.doJSON(ctx, http.MethodPost, "/v1/taskhubs/orchestrations/addhistoryevents", req, &unsuccessful); err != nil {
		return nil, err
	}
	return unsuccessful, nil
}

// helper: split string at last occurrence of sep
func splitLast(s, sep string) []string {
	idx := -1
	for i := len(s) - 1; i >= 0; i-- {
		if string(s[i]) == sep {
			idx = i
			break
		}
	}
	if idx < 0 {
		return []string{s}
	}
	return []string{s[:idx], s[idx+1:]}
}

// helper: read response body as string
func readBody(resp *http.Response) (string, int, error) {
	data, err := readAll(resp)
	if err != nil {
		return "", resp.StatusCode, err
	}
	return string(data), resp.StatusCode, nil
}

func readAll(resp *http.Response) ([]byte, error) {
	data := make([]byte, 0, 1024)
	buf := make([]byte, 512)
	for {
		n, err := resp.Body.Read(buf)
		if n > 0 {
			data = append(data, buf[:n]...)
		}
		if err != nil {
			break
		}
	}
	return data, nil
}

func decodeJSON(resp *http.Response, v interface{}) error {
	data, err := readAll(resp)
	if err != nil {
		return err
	}

	// Try direct decode
	if err := jsonUnmarshal(data, v); err != nil {
		return fmt.Errorf("json decode: %w (body: %s)", err, truncateStr(string(data), 200))
	}
	return nil
}

func jsonUnmarshal(data []byte, v interface{}) error {
	// use standard json.Unmarshal
	return unmarshalJSON(data, v)
}

// unmarshalJSON is a simple wrapper using encoding/json.
func unmarshalJSON(data []byte, v interface{}) error {
	return jsonDecode(data, v)
}

func jsonDecode(data []byte, v interface{}) error {
	// Direct implementation using encoding/json
	dec := jsonNewDecoder(data)
	return dec.Decode(v)
}

func jsonNewDecoder(data []byte) *jsonDecWrapper {
	return &jsonDecWrapper{data: data}
}

type jsonDecWrapper struct {
	data []byte
}

func (d *jsonDecWrapper) Decode(v interface{}) error {
	return json.Unmarshal(d.data, v)
}

func truncateStr(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max] + "..."
}
