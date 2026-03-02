// Package api provides HTTP client types and request/response DTOs for the DTS Backend API.
package api

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"
)

// --- Orchestration Models ---

// Orchestration represents an orchestration instance metadata.
type Orchestration struct {
	InstanceID              string            `json:"instanceId"`
	ExecutionID             string            `json:"executionId,omitempty"`
	Name                    string            `json:"name"`
	Version                 string            `json:"version,omitempty"`
	CreatedTimestamp        time.Time         `json:"createdTimestamp"`
	LastUpdatedTimestamp    time.Time         `json:"lastUpdatedTimestamp"`
	CompletedTimestamp      *time.Time        `json:"completedTimestamp,omitempty"`
	OrchestrationStatus     string            `json:"orchestrationStatus"`
	ScheduledStartTimestamp *time.Time        `json:"scheduledStartTimestamp,omitempty"`
	ParentInstanceID        string            `json:"parentInstanceId,omitempty"`
	Tags                    map[string]string `json:"tags,omitempty"`
}

// OrchestrationPayloads contains the input/output/failure data for an orchestration.
type OrchestrationPayloads struct {
	Input               string          `json:"input,omitempty"`
	Output              string          `json:"output,omitempty"`
	CustomStatus        string          `json:"customStatus,omitempty"`
	OrchestrationStatus string          `json:"orchestrationStatus"`
	FailureDetails      *FailureDetails `json:"failureDetails,omitempty"`
}

// FailureDetails describes an orchestration failure.
type FailureDetails struct {
	ErrorType      string          `json:"errorType"`
	ErrorMessage   string          `json:"errorMessage"`
	StackTrace     string          `json:"stackTrace,omitempty"`
	InnerFailure   *FailureDetails `json:"innerFailure,omitempty"`
	IsNonRetriable bool            `json:"isNonRetriable"`
}

// OrchestrationsResult is the response from querying orchestrations.
type OrchestrationsResult struct {
	Orchestrations []Orchestration      `json:"orchestrations"`
	TotalCount     int                  `json:"totalCount"`
	Trivia         *OrchestrationTrivia `json:"trivia,omitempty"`
}

// OrchestrationTrivia contains summary statistics.
type OrchestrationTrivia struct {
	EarliestTimestamp *time.Time `json:"earliestTimestamp,omitempty"`
	LatestTimestamp   *time.Time `json:"latestTimestamp,omitempty"`
	TotalCount        int        `json:"totalCount"`
	CompletedCount    int        `json:"completedCount"`
	RunningCount      int        `json:"runningCount"`
	FailedCount       int        `json:"failedCount"`
	PendingCount      int        `json:"pendingCount"`
}

// --- Query Models ---

// QueryOrchestrationsRequest is the request body for POST /orchestrations/query.
type QueryOrchestrationsRequest struct {
	Filter     *OrchestrationFilter `json:"filter,omitempty"`
	Pagination *Pagination          `json:"pagination,omitempty"`
	Sort       []SortOption         `json:"sort,omitempty"`
	Fields     string               `json:"fields,omitempty"`
}

// OrchestrationFilter specifies filters for orchestration queries.
type OrchestrationFilter struct {
	OrchestrationID     *StringFilter   `json:"orchestrationId,omitempty"`
	Name                *StringFilter   `json:"name,omitempty"`
	Version             *StringFilter   `json:"version,omitempty"`
	CreatedAt           *DateTimeFilter `json:"createdAt,omitempty"`
	LastUpdatedAt       *DateTimeFilter `json:"lastUpdatedAt,omitempty"`
	StartAt             *DateTimeFilter `json:"startAt,omitempty"`
	CompletedAt         *DateTimeFilter `json:"completedAt,omitempty"`
	OrchestrationStatus *StatusFilter   `json:"orchestrationStatus,omitempty"`
	Tags                *StringFilter   `json:"tags,omitempty"`
}

// StringFilter matches a case-insensitive substring.
type StringFilter struct {
	Value string `json:"value"`
}

// DateTimeFilter specifies a time range (start inclusive, end exclusive).
type DateTimeFilter struct {
	Start *time.Time `json:"start,omitempty"`
	End   *time.Time `json:"end,omitempty"`
}

// StatusFilter specifies one or more orchestration statuses.
type StatusFilter struct {
	Status []string `json:"status"`
}

// Pagination specifies offset-based pagination.
type Pagination struct {
	StartIndex int `json:"startIndex"`
	Count      int `json:"count"`
}

// SortOption specifies a sort column and direction.
type SortOption struct {
	Column    string `json:"column"`
	Direction string `json:"direction"`
}

// --- Create Orchestration ---

// CreateOrchestrationRequest is the request body for POST /orchestrations.
type CreateOrchestrationRequest struct {
	Name                    string            `json:"name"`
	InstanceID              string            `json:"instanceId,omitempty"`
	ExecutionID             string            `json:"executionId,omitempty"`
	Input                   string            `json:"input,omitempty"`
	Version                 string            `json:"version,omitempty"`
	ScheduledStartTimestamp *time.Time        `json:"scheduledStartTimestamp,omitempty"`
	Tags                    map[string]string `json:"tags,omitempty"`
}

// RestartOrchestrationRequest is the request body for POST /orchestrations/{id}/restart.
type RestartOrchestrationRequest struct {
	RestartWithNewInstanceID bool `json:"restartWithNewInstanceId"`
}

// ForceTerminateRequest is the request body for POST /orchestrations/forceterminate.
type ForceTerminateRequest struct {
	InstanceIDs []string `json:"InstanceIds"`
	Reason      string   `json:"Reason,omitempty"`
}

// BatchHistoryEventsRequest is the request body for POST /orchestrations/addhistoryevents.
type BatchHistoryEventsRequest struct {
	Events map[string]interface{} `json:"events"`
}

// --- History Event Models ---

// HistoryEvent represents a single orchestration history event in protobuf-JSON format.
// Each event contains common metadata (eventId, timestamp) plus exactly one
// oneof field identifying the event type with its type-specific data.
type HistoryEvent struct {
	EventID   *int   `json:"eventId,omitempty"`
	Timestamp string `json:"timestamp,omitempty"`

	// Protobuf oneof event type fields — exactly one will be non-nil.
	ExecutionStarted                  *ExecutionStartedEvent    `json:"executionStarted,omitempty"`
	ExecutionCompleted                *ExecutionCompletedEvent  `json:"executionCompleted,omitempty"`
	ExecutionFailed                   *ExecutionFailedEvent     `json:"executionFailed,omitempty"`
	ExecutionTerminated               *ExecutionTerminatedEvent `json:"executionTerminated,omitempty"`
	TaskScheduled                     *TaskScheduledEvent       `json:"taskScheduled,omitempty"`
	TaskCompleted                     *TaskCompletedEvent       `json:"taskCompleted,omitempty"`
	TaskFailed                        *TaskFailedEvent          `json:"taskFailed,omitempty"`
	SubOrchestrationInstanceCreated   *SubOrchCreatedEvent      `json:"subOrchestrationInstanceCreated,omitempty"`
	SubOrchestrationInstanceCompleted *SubOrchCompletedEvent    `json:"subOrchestrationInstanceCompleted,omitempty"`
	SubOrchestrationInstanceFailed    *SubOrchFailedEvent       `json:"subOrchestrationInstanceFailed,omitempty"`
	TimerCreated                      *TimerCreatedEvent        `json:"timerCreated,omitempty"`
	TimerFired                        *TimerFiredEvent          `json:"timerFired,omitempty"`
	EventRaised                       *EventRaisedEvent         `json:"eventRaised,omitempty"`
	EventSent                         *EventSentEvent           `json:"eventSent,omitempty"`
	ExecutionSuspended                *ExecutionSuspendedEvent  `json:"executionSuspended,omitempty"`
	ExecutionResumed                  *ExecutionResumedEvent    `json:"executionResumed,omitempty"`
}

// Type returns the event type name (e.g., "ExecutionStarted", "TaskScheduled").
func (e *HistoryEvent) Type() string {
	switch {
	case e.ExecutionStarted != nil:
		return "ExecutionStarted"
	case e.ExecutionCompleted != nil:
		return "ExecutionCompleted"
	case e.ExecutionFailed != nil:
		return "ExecutionFailed"
	case e.ExecutionTerminated != nil:
		return "ExecutionTerminated"
	case e.TaskScheduled != nil:
		return "TaskScheduled"
	case e.TaskCompleted != nil:
		return "TaskCompleted"
	case e.TaskFailed != nil:
		return "TaskFailed"
	case e.SubOrchestrationInstanceCreated != nil:
		return "SubOrchestrationInstanceCreated"
	case e.SubOrchestrationInstanceCompleted != nil:
		return "SubOrchestrationInstanceCompleted"
	case e.SubOrchestrationInstanceFailed != nil:
		return "SubOrchestrationInstanceFailed"
	case e.TimerCreated != nil:
		return "TimerCreated"
	case e.TimerFired != nil:
		return "TimerFired"
	case e.EventRaised != nil:
		return "EventRaised"
	case e.EventSent != nil:
		return "EventSent"
	case e.ExecutionSuspended != nil:
		return "ExecutionSuspended"
	case e.ExecutionResumed != nil:
		return "ExecutionResumed"
	default:
		return "Unknown"
	}
}

// ParseTimestamp parses the event timestamp.
func (e *HistoryEvent) ParseTimestamp() time.Time {
	if e.Timestamp == "" {
		return time.Time{}
	}
	if t, err := time.Parse(time.RFC3339Nano, e.Timestamp); err == nil {
		return t
	}
	return time.Time{}
}

// EventName returns the name associated with this event (e.g., activity/orchestration name).
func (e *HistoryEvent) EventName() string {
	switch {
	case e.ExecutionStarted != nil:
		return e.ExecutionStarted.Name
	case e.TaskScheduled != nil:
		return e.TaskScheduled.Name
	case e.SubOrchestrationInstanceCreated != nil:
		return e.SubOrchestrationInstanceCreated.Name
	case e.EventRaised != nil:
		return e.EventRaised.Name
	case e.EventSent != nil:
		return e.EventSent.Name
	default:
		return ""
	}
}

// ScheduledID returns the TaskScheduledId for completion/failure events, or -1.
func (e *HistoryEvent) ScheduledID() int {
	switch {
	case e.TaskCompleted != nil:
		return e.TaskCompleted.TaskScheduledID
	case e.TaskFailed != nil:
		return e.TaskFailed.TaskScheduledID
	case e.SubOrchestrationInstanceCompleted != nil:
		return e.SubOrchestrationInstanceCompleted.TaskScheduledID
	case e.SubOrchestrationInstanceFailed != nil:
		return e.SubOrchestrationInstanceFailed.TaskScheduledID
	default:
		return -1
	}
}

// EventIDValue returns the event ID as an int, or -1 if nil.
func (e *HistoryEvent) EventIDValue() int {
	if e.EventID != nil {
		return *e.EventID
	}
	return -1
}

// EventIDString returns the event ID formatted for display.
// Returns "" for nil (omitted/null) event IDs instead of "-1".
func (e *HistoryEvent) EventIDString() string {
	if e.EventID != nil {
		return fmt.Sprintf("%d", *e.EventID)
	}
	return ""
}

// DisplayID returns the most meaningful identifier for display in the history table.
// For completion/failure events it returns the nested taskScheduledId or timerId
// that correlates the event back to its "scheduled" counterpart, matching the
// behaviour of the DTS web dashboard.
func (e *HistoryEvent) DisplayID() string {
	switch {
	// Completion/failure events → show the nested scheduledId they reference
	case e.TaskCompleted != nil:
		return fmt.Sprintf("%d", e.TaskCompleted.TaskScheduledID)
	case e.TaskFailed != nil:
		return fmt.Sprintf("%d", e.TaskFailed.TaskScheduledID)
	case e.SubOrchestrationInstanceCompleted != nil:
		return fmt.Sprintf("%d", e.SubOrchestrationInstanceCompleted.TaskScheduledID)
	case e.SubOrchestrationInstanceFailed != nil:
		return fmt.Sprintf("%d", e.SubOrchestrationInstanceFailed.TaskScheduledID)
	// TimerFired → show the timerId it references
	case e.TimerFired != nil:
		return fmt.Sprintf("%d", e.TimerFired.TimerID)
	// Scheduled/created events and others → use the top-level eventId
	default:
		return e.EventIDString()
	}
}

// FiredTimerID returns the TimerId from a TimerFired event, or -1.
func (e *HistoryEvent) FiredTimerID() int {
	if e.TimerFired != nil {
		return e.TimerFired.TimerID
	}
	return -1
}

// FormatTags returns a formatted string of event tags, or "".
func (e *HistoryEvent) FormatTags() string {
	var tags map[string]string
	switch {
	case e.ExecutionStarted != nil:
		tags = e.ExecutionStarted.Tags
	case e.TaskScheduled != nil:
		tags = e.TaskScheduled.Tags
	case e.SubOrchestrationInstanceCreated != nil:
		tags = e.SubOrchestrationInstanceCreated.Tags
	}
	if len(tags) == 0 {
		return ""
	}
	parts := make([]string, 0, len(tags))
	for k, v := range tags {
		parts = append(parts, k+"="+v)
	}
	sort.Strings(parts)
	return strings.Join(parts, ", ")
}

// --- History Event Sub-Types ---

// ExecutionStartedEvent contains data for an ExecutionStarted history event.
type ExecutionStartedEvent struct {
	Name                    string                 `json:"name,omitempty"`
	Version                 string                 `json:"version,omitempty"`
	Input                   json.RawMessage        `json:"input,omitempty"`
	Tags                    map[string]string      `json:"tags,omitempty"`
	OrchestrationInstance   *OrchestrationInstance `json:"orchestrationInstance,omitempty"`
	ParentInstance          *OrchestrationInstance `json:"parentInstance,omitempty"`
	ScheduledStartTimestamp string                 `json:"scheduledStartTimestamp,omitempty"`
}

// OrchestrationInstance identifies an orchestration instance.
type OrchestrationInstance struct {
	InstanceID  string `json:"instanceId,omitempty"`
	ExecutionID string `json:"executionId,omitempty"`
}

// ExecutionCompletedEvent contains data for an ExecutionCompleted history event.
type ExecutionCompletedEvent struct {
	OrchestrationStatus string          `json:"orchestrationStatus,omitempty"`
	Result              json.RawMessage `json:"result,omitempty"`
}

// ExecutionFailedEvent contains data for an ExecutionFailed history event.
type ExecutionFailedEvent struct {
	FailureDetails *FailureDetails `json:"failureDetails,omitempty"`
}

// ExecutionTerminatedEvent contains data for an ExecutionTerminated history event.
type ExecutionTerminatedEvent struct {
	Input  json.RawMessage `json:"input,omitempty"`
	Reason string          `json:"reason,omitempty"`
}

// TaskScheduledEvent contains data for a TaskScheduled history event.
type TaskScheduledEvent struct {
	Name    string            `json:"name,omitempty"`
	Version string            `json:"version,omitempty"`
	Input   json.RawMessage   `json:"input,omitempty"`
	Tags    map[string]string `json:"tags,omitempty"`
}

// TaskCompletedEvent contains data for a TaskCompleted history event.
type TaskCompletedEvent struct {
	TaskScheduledID int             `json:"taskScheduledId"`
	Result          json.RawMessage `json:"result,omitempty"`
}

// TaskFailedEvent contains data for a TaskFailed history event.
type TaskFailedEvent struct {
	TaskScheduledID int             `json:"taskScheduledId"`
	FailureDetails  *FailureDetails `json:"failureDetails,omitempty"`
}

// SubOrchCreatedEvent contains data for a SubOrchestrationInstanceCreated history event.
type SubOrchCreatedEvent struct {
	Name       string            `json:"name,omitempty"`
	Version    string            `json:"version,omitempty"`
	Input      json.RawMessage   `json:"input,omitempty"`
	InstanceID string            `json:"instanceId,omitempty"`
	Tags       map[string]string `json:"tags,omitempty"`
}

// SubOrchCompletedEvent contains data for a SubOrchestrationInstanceCompleted history event.
type SubOrchCompletedEvent struct {
	TaskScheduledID int             `json:"taskScheduledId"`
	Result          json.RawMessage `json:"result,omitempty"`
}

// SubOrchFailedEvent contains data for a SubOrchestrationInstanceFailed history event.
type SubOrchFailedEvent struct {
	TaskScheduledID int             `json:"taskScheduledId"`
	FailureDetails  *FailureDetails `json:"failureDetails,omitempty"`
}

// TimerCreatedEvent contains data for a TimerCreated history event.
type TimerCreatedEvent struct {
	FireAt string `json:"fireAt,omitempty"`
}

// TimerFiredEvent contains data for a TimerFired history event.
type TimerFiredEvent struct {
	TimerID int    `json:"timerId"`
	FireAt  string `json:"fireAt,omitempty"`
}

// EventRaisedEvent contains data for an EventRaised history event.
type EventRaisedEvent struct {
	Name  string          `json:"name,omitempty"`
	Input json.RawMessage `json:"input,omitempty"`
}

// EventSentEvent contains data for an EventSent history event.
type EventSentEvent struct {
	Name       string          `json:"name,omitempty"`
	InstanceID string          `json:"instanceId,omitempty"`
	Input      json.RawMessage `json:"input,omitempty"`
}

// ExecutionSuspendedEvent contains data for an ExecutionSuspended history event.
type ExecutionSuspendedEvent struct {
	Reason string `json:"reason,omitempty"`
}

// ExecutionResumedEvent contains data for an ExecutionResumed history event.
type ExecutionResumedEvent struct {
	Reason string `json:"reason,omitempty"`
}

// --- Entity Models ---

// Entity represents an entity instance.
type Entity struct {
	InstanceID       string    `json:"instanceId"`
	LastModifiedTime time.Time `json:"lastModifiedTime"`
	BacklogQueueSize int       `json:"backlogQueueSize"`
	LockedBy         string    `json:"lockedBy,omitempty"`
}

// EntityName extracts the entity name from the instance ID (format: @entity@Name@Key).
func (e *Entity) EntityName() string {
	return extractEntityPart(e.InstanceID, 1)
}

// EntityKey extracts the entity key from the instance ID.
func (e *Entity) EntityKey() string {
	return extractEntityPart(e.InstanceID, 2)
}

func extractEntityPart(instanceID string, index int) string {
	// Format: @entity@Name@Key or @<type>@Name@Key
	count := 0
	start := 0
	for i := 0; i < len(instanceID); i++ {
		if instanceID[i] == '@' {
			if count == index {
				start = i + 1
			} else if count == index+1 {
				return instanceID[start:i]
			}
			count++
		}
	}
	if count == index+1 && start > 0 {
		return instanceID[start:]
	}
	if index == 0 {
		return instanceID
	}
	return ""
}

// ParseEntityName extracts the entity name from an instance ID string.
func ParseEntityName(instanceID string) string {
	return extractEntityPart(instanceID, 1)
}

// ParseEntityKey extracts the entity key from an instance ID string.
func ParseEntityKey(instanceID string) string {
	return extractEntityPart(instanceID, 2)
}

// QueryEntitiesRequest is the request body for POST /entities/query.
type QueryEntitiesRequest struct {
	Filter          *EntityFilter `json:"filter,omitempty"`
	Pagination      *Pagination   `json:"pagination,omitempty"`
	Sort            []SortOption  `json:"sort,omitempty"`
	FetchTotalCount bool          `json:"fetchTotalCount"`
}

// EntityFilter specifies filters for entity queries.
type EntityFilter struct {
	Key                   *StringFilter   `json:"key,omitempty"`
	Name                  *StringFilter   `json:"name,omitempty"`
	NameStartsWith        *StringFilter   `json:"nameStartsWith,omitempty"`
	ExcludeNameStartsWith []StringFilter  `json:"excludeNameStartsWith,omitempty"`
	LastModifiedAt        *DateTimeFilter `json:"lastModifiedAt,omitempty"`
}

// QueryEntitiesResult is the response from querying entities.
type QueryEntitiesResult struct {
	Entities   []Entity `json:"entities"`
	TotalCount int      `json:"totalCount"`
}

// EntitiesResult is the response from listing entities (with continuation token).
type EntitiesResult struct {
	Entities          []Entity `json:"entities"`
	ContinuationToken string   `json:"continuationToken,omitempty"`
}

// --- Schedule Models ---

// Schedule represents a scheduled orchestration.
type Schedule struct {
	ScheduleConfiguration ScheduleConfiguration `json:"ScheduleConfiguration"`
	Status                int                   `json:"Status"`
	ExecutionToken        string                `json:"ExecutionToken,omitempty"`
	LastRunAt             *time.Time            `json:"LastRunAt,omitempty"`
	LastRunStatus         *string               `json:"LastRunStatus,omitempty"`
	ScheduleCreatedAt     *time.Time            `json:"ScheduleCreatedAt,omitempty"`
	ScheduleLastModified  *time.Time            `json:"ScheduleLastModifiedAt,omitempty"`
}

// ScheduleConfiguration holds the schedule parameters.
type ScheduleConfiguration struct {
	ScheduleID              string     `json:"ScheduleId"`
	OrchestrationName       string     `json:"OrchestrationName"`
	OrchestrationInput      string     `json:"OrchestrationInput,omitempty"`
	OrchestrationInstanceID string     `json:"OrchestrationInstanceId,omitempty"`
	Interval                string     `json:"Interval,omitempty"`
	StartAt                 *time.Time `json:"StartAt,omitempty"`
	EndAt                   *time.Time `json:"EndAt,omitempty"`
	StartImmediatelyIfLate  bool       `json:"StartImmediatelyIfLate"`
}

// SchedulesResult is the response from listing schedules.
type SchedulesResult struct {
	Entities          []Schedule `json:"entities"`
	ContinuationToken string     `json:"continuationToken,omitempty"`
}

// CreateScheduleRequest is the request body for POST /schedules.
type CreateScheduleRequest struct {
	ScheduleID              string     `json:"ScheduleId"`
	OrchestrationName       string     `json:"OrchestrationName"`
	Interval                string     `json:"Interval"`
	OrchestrationInput      string     `json:"OrchestrationInput,omitempty"`
	OrchestrationInstanceID string     `json:"OrchestrationInstanceId,omitempty"`
	StartAt                 *time.Time `json:"StartAt,omitempty"`
	EndAt                   *time.Time `json:"EndAt,omitempty"`
	StartImmediatelyIfLate  bool       `json:"StartImmediatelyIfLate"`
}

// --- Worker Models ---

// Worker represents a connected worker.
type Worker struct {
	WorkerID                  string `json:"workerId"`
	ActiveOrchestrationsCount int    `json:"activeOrchestrationsCount"`
	MaxOrchestrationsCount    int    `json:"maxOrchestrationsCount"`
	ActiveActivitiesCount     int    `json:"activeActivitiesCount"`
	MaxActivitiesCount        int    `json:"maxActivitiesCount"`
	ActiveEntitiesCount       int    `json:"activeEntitiesCount"`
	MaxEntitiesCount          int    `json:"maxEntitiesCount"`
}

// WorkersResult is the response from listing workers.
type WorkersResult struct {
	Workers []Worker `json:"workers"`
}

// --- Agent Models ---

// AgentEntity represents an agent session entity.
type AgentEntity struct {
	Name         string    `json:"name"`
	SessionID    string    `json:"sessionId"`
	EntityID     string    `json:"entityId"`
	LastModified time.Time `json:"lastModifiedTime"`
}

// AgentState represents the durable state of an agent session.
type AgentState struct {
	Status   string         `json:"status,omitempty"`
	Requests []AgentRequest `json:"requests"`
	Messages []AgentMessage `json:"messages,omitempty"` // flattened view
}

// AgentRequest represents a single request/response in the agent conversation.
type AgentRequest struct {
	RequestMessages  []AgentMessage `json:"requestMessages"`
	ResponseMessages []AgentMessage `json:"responseMessages"`
	Timestamp        *time.Time     `json:"timestamp,omitempty"`
	TotalTokens      int            `json:"totalTokens,omitempty"`
}

// AgentMessage is a single message in the agent conversation.
type AgentMessage struct {
	Role         string             `json:"role"`
	Content      string             `json:"content,omitempty"`
	FunctionCall *AgentFunctionCall `json:"functionCall,omitempty"`
	Timestamp    *time.Time         `json:"timestamp,omitempty"`
}

// AgentFunctionCall represents a function/tool call in the agent conversation.
type AgentFunctionCall struct {
	Name      string `json:"name"`
	Arguments string `json:"arguments,omitempty"`
	Result    string `json:"result,omitempty"`
}

// --- Sort/Filter Constants ---

const (
	// Sort columns for orchestrations
	SortByOrchestrationID = "ORCHESTRATION_ID"
	SortByName            = "NAME"
	SortByVersion         = "VERSION"
	SortByCreatedAt       = "CREATED_AT"
	SortByLastUpdatedAt   = "LAST_UPDATED_AT"
	SortByStartAt         = "START_AT"
	SortByCompletedAt     = "COMPLETED_AT"
	SortByStatus          = "ORCHESTRATION_STATUS"
	SortByTags            = "TAGS"

	// Sort columns for entities
	SortEntityByName           = "NAME"
	SortEntityByKey            = "KEY"
	SortEntityByLastModifiedAt = "LAST_MODIFIED_AT"

	// Sort directions
	SortAscending  = "ASCENDING_SORT"
	SortDescending = "DESCENDING_SORT"

	// Orchestration statuses
	StatusRunning        = "ORCHESTRATION_STATUS_RUNNING"
	StatusCompleted      = "ORCHESTRATION_STATUS_COMPLETED"
	StatusFailed         = "ORCHESTRATION_STATUS_FAILED"
	StatusPending        = "ORCHESTRATION_STATUS_PENDING"
	StatusSuspended      = "ORCHESTRATION_STATUS_SUSPENDED"
	StatusTerminated     = "ORCHESTRATION_STATUS_TERMINATED"
	StatusCanceled       = "ORCHESTRATION_STATUS_CANCELED"
	StatusContinuedAsNew = "ORCHESTRATION_STATUS_CONTINUED_AS_NEW"

	// Default fields for orchestration list queries
	DefaultOrchestrationFields = "instanceId,name,version,createdTimestamp,executionId,lastUpdatedTimestamp,completedTimestamp,orchestrationStatus,scheduledStartTimestamp,tags"
)

// IsTerminal returns true if the orchestration status represents a terminal state.
func IsTerminal(status string) bool {
	switch status {
	case StatusCompleted, StatusFailed, StatusTerminated, StatusCanceled:
		return true
	}
	return false
}

// CanSuspend returns true if the orchestration can be suspended.
func CanSuspend(status string) bool {
	return status == StatusRunning || status == StatusPending
}

// CanResume returns true if the orchestration can be resumed.
func CanResume(status string) bool {
	return status == StatusSuspended
}

// CanRewind returns true if the orchestration can be rewound.
func CanRewind(status string) bool {
	return status == StatusFailed
}

// CanTerminate returns true if the orchestration can be terminated.
func CanTerminate(status string) bool {
	return !IsTerminal(status)
}

// CanPurge returns true if the orchestration can be purged.
func CanPurge(status string) bool {
	return IsTerminal(status) || status == StatusSuspended
}
