package api

import (
	"context"
	"fmt"
	"net/http"
)

// ListSchedules lists schedules with optional continuation token.
func (c *Client) ListSchedules(ctx context.Context, continuationToken string) (*SchedulesResult, error) {
	path := "/v1/taskhubs/schedules"
	if continuationToken != "" {
		path += "?continuationToken=" + continuationToken
	}
	var result SchedulesResult
	if err := c.doJSON(ctx, http.MethodGet, path, nil, &result); err != nil {
		return nil, fmt.Errorf("listing schedules: %w", err)
	}
	return &result, nil
}

// CreateSchedule creates a new schedule.
func (c *Client) CreateSchedule(ctx context.Context, req *CreateScheduleRequest) error {
	if err := c.doNoContent(ctx, http.MethodPost, "/v1/taskhubs/schedules", req); err != nil {
		return fmt.Errorf("creating schedule: %w", err)
	}
	return nil
}

// DeleteSchedule deletes a schedule.
func (c *Client) DeleteSchedule(ctx context.Context, scheduleID string) error {
	path := fmt.Sprintf("/v1/taskhubs/schedules/%s", scheduleID)
	if err := c.doNoContent(ctx, http.MethodDelete, path, nil); err != nil {
		return fmt.Errorf("deleting schedule %s: %w", scheduleID, err)
	}
	return nil
}

// PauseSchedule pauses a schedule.
func (c *Client) PauseSchedule(ctx context.Context, scheduleID string) error {
	path := fmt.Sprintf("/v1/taskhubs/schedules/%s/pause", scheduleID)
	if err := c.doNoContent(ctx, http.MethodPost, path, nil); err != nil {
		return fmt.Errorf("pausing schedule %s: %w", scheduleID, err)
	}
	return nil
}

// ResumeSchedule resumes a paused schedule.
func (c *Client) ResumeSchedule(ctx context.Context, scheduleID string) error {
	path := fmt.Sprintf("/v1/taskhubs/schedules/%s/resume", scheduleID)
	if err := c.doNoContent(ctx, http.MethodPost, path, nil); err != nil {
		return fmt.Errorf("resuming schedule %s: %w", scheduleID, err)
	}
	return nil
}
