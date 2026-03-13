package api

import (
	"context"
	"fmt"
	"net/http"
)

// ListWorkers lists all connected workers.
func (c *Client) ListWorkers(ctx context.Context) (*WorkersResult, error) {
	var result WorkersResult
	if err := c.doJSON(ctx, http.MethodGet, "/v1/taskhubs/workers", nil, &result); err != nil {
		return nil, fmt.Errorf("listing workers: %w", err)
	}
	return &result, nil
}
