package api

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
)

// ListAgentSessions lists agent sessions by querying entities with agent name prefix.
func (c *Client) ListAgentSessions(ctx context.Context, pageSize int, startIndex int) (*QueryEntitiesResult, error) {
	req := &QueryEntitiesRequest{
		Filter: &EntityFilter{
			NameStartsWith: &StringFilter{Value: "@agent@"},
		},
		Pagination: &Pagination{
			StartIndex: startIndex,
			Count:      pageSize,
		},
		Sort: []SortOption{
			{Column: SortEntityByLastModifiedAt, Direction: SortDescending},
		},
		FetchTotalCount: true,
	}
	return c.QueryEntities(ctx, req)
}

// GetAgentState gets the deserialized state of an agent session.
func (c *Client) GetAgentState(ctx context.Context, name, sessionID string) (*AgentState, error) {
	instanceID := fmt.Sprintf("@agent@%s@%s", name, sessionID)
	raw, err := c.GetEntityState(ctx, instanceID)
	if err != nil {
		return nil, fmt.Errorf("getting agent state: %w", err)
	}

	var state AgentState
	if err := json.Unmarshal([]byte(raw), &state); err != nil {
		return nil, fmt.Errorf("parsing agent state: %w", err)
	}
	return &state, nil
}

// StartAgentSession starts a new agent session by creating an orchestration
// that signals the agent entity.
func (c *Client) StartAgentSession(ctx context.Context, agentName, sessionID, prompt string) (string, error) {
	req := &CreateOrchestrationRequest{
		Name:       agentName,
		InstanceID: fmt.Sprintf("agent-%s-%s", agentName, sessionID),
		Input:      fmt.Sprintf(`{"sessionId":"%s","prompt":"%s"}`, sessionID, escapeJSON(prompt)),
	}
	return c.CreateOrchestration(ctx, req)
}

// SendAgentPrompt sends an event to an existing agent session's orchestration.
func (c *Client) SendAgentPrompt(ctx context.Context, agentName, sessionID, prompt string) error {
	instanceID := fmt.Sprintf("agent-%s-%s", agentName, sessionID)
	return c.RaiseEvent(ctx, instanceID, "UserPrompt", fmt.Sprintf(`{"prompt":"%s"}`, escapeJSON(prompt)))
}

// DeleteAgentSession deletes an agent entity by its instance ID.
func (c *Client) DeleteAgentSession(ctx context.Context, instanceID string) error {
	return c.DeleteEntity(ctx, instanceID)
}

// DeleteAgentSessions deletes multiple agent entities.
func (c *Client) DeleteAgentSessions(ctx context.Context, instanceIDs []string) error {
	return c.DeleteEntities(ctx, instanceIDs)
}

// ParseAgentEntity extracts name and sessionID from an agent entity.
func ParseAgentEntity(entity *Entity) *AgentEntity {
	// Instance ID format: @agent@Name@SessionId
	parts := strings.SplitN(entity.InstanceID, "@", 5)
	// parts: ["", "agent", "Name", "SessionId"] or similar
	name := ""
	sessionID := ""
	if len(parts) >= 4 {
		name = parts[2]
		sessionID = strings.Join(parts[3:], "@") // session ID may contain @
	}
	return &AgentEntity{
		Name:         name,
		SessionID:    sessionID,
		EntityID:     entity.InstanceID,
		LastModified: entity.LastModifiedTime,
	}
}

func escapeJSON(s string) string {
	b, _ := json.Marshal(s)
	// Remove surrounding quotes from json.Marshal
	if len(b) >= 2 {
		return string(b[1 : len(b)-1])
	}
	return s
}
