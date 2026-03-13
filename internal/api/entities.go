package api

import (
	"context"
	"fmt"
	"net/http"
)

// QueryEntities queries entities with filtering, sorting, and pagination.
func (c *Client) QueryEntities(ctx context.Context, req *QueryEntitiesRequest) (*QueryEntitiesResult, error) {
	var result QueryEntitiesResult
	if err := c.doJSON(ctx, http.MethodPost, "/v1/taskhubs/entities/query", req, &result); err != nil {
		return nil, fmt.Errorf("querying entities: %w", err)
	}
	return &result, nil
}

// GetEntity gets a single entity by instance ID.
func (c *Client) GetEntity(ctx context.Context, instanceID string) (*Entity, error) {
	var result Entity
	path := fmt.Sprintf("/v1/taskhubs/entities/%s", instanceID)
	if err := c.doJSON(ctx, http.MethodGet, path, nil, &result); err != nil {
		return nil, fmt.Errorf("getting entity %s: %w", instanceID, err)
	}
	return &result, nil
}

// GetEntityState gets the serialized state of an entity.
func (c *Client) GetEntityState(ctx context.Context, instanceID string) (string, error) {
	path := fmt.Sprintf("/v1/taskhubs/entities/%s/state", instanceID)
	body, _, err := c.doRaw(ctx, http.MethodGet, path, nil)
	if err != nil {
		return "", fmt.Errorf("getting entity state %s: %w", instanceID, err)
	}
	return body, nil
}

// DeleteEntity deletes a single entity.
func (c *Client) DeleteEntity(ctx context.Context, instanceID string) error {
	path := fmt.Sprintf("/v1/taskhubs/entities/%s", instanceID)
	if err := c.doNoContent(ctx, http.MethodDelete, path, nil); err != nil {
		return fmt.Errorf("deleting entity %s: %w", instanceID, err)
	}
	return nil
}

// DeleteEntities deletes multiple entities.
func (c *Client) DeleteEntities(ctx context.Context, instanceIDs []string) error {
	if err := c.doNoContent(ctx, http.MethodDelete, "/v1/taskhubs/entities/delete", instanceIDs); err != nil {
		return fmt.Errorf("deleting entities: %w", err)
	}
	return nil
}
