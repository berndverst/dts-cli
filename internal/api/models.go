// Package api provides HTTP client types and request/response DTOs for the DTS Backend API.
package api

import "time"

// --- Orchestration Models ---

// Orchestration represents an orchestration instance metadata.
type Orchestration struct {
	InstanceID            string            `json:"instanceId"`
	ExecutionID           string            `json:"executionId,omitempty"`
	Name                  string            `json:"name"`
	Version               string            `json:"version,omitempty"`
	CreatedTimestamp       time.Time         `json:"createdTimestamp"`
	LastUpdatedTimestamp   time.Time         `json:"lastUpdatedTimestamp"`
	CompletedTimestamp     *time.Time        `json:"completedTimestamp,omitempty"`
	OrchestrationStatus   string            `json:"orchestrationStatus"`
	ScheduledStartTimestamp *time.Time       `json:"scheduledStartTimestamp,omitempty"`
	ParentInstanceID      string            `json:"parentInstanceId,omitempty"`
	Tags                  map[string]string `json:"tags,omitempty"`
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
	OrchestrationID     *StringFilter         `json:"orchestrationId,omitempty"`
	Name                *StringFilter         `json:"name,omitempty"`
	Version             *StringFilter         `json:"version,omitempty"`
	CreatedAt           *DateTimeFilter       `json:"createdAt,omitempty"`
	LastUpdatedAt       *DateTimeFilter       `json:"lastUpdatedAt,omitempty"`
	StartAt             *DateTimeFilter       `json:"startAt,omitempty"`
	CompletedAt         *DateTimeFilter       `json:"completedAt,omitempty"`
	OrchestrationStatus *StatusFilter         `json:"orchestrationStatus,omitempty"`
	Tags                *StringFilter         `json:"tags,omitempty"`
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

// HistoryEvent represents a single orchestration history event.
// The event is a protobuf union; we use map for flexibility.
type HistoryEvent map[string]interface{}

// --- Entity Models ---

// Entity represents an entity instance.
type Entity struct {
	InstanceID       string     `json:"instanceId"`
	LastModifiedTime time.Time  `json:"lastModifiedTime"`
	BacklogQueueSize int        `json:"backlogQueueSize"`
	LockedBy         string     `json:"lockedBy,omitempty"`
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
	Status                int                    `json:"Status"`
	ExecutionToken        string                 `json:"ExecutionToken,omitempty"`
	LastRunAt             *time.Time             `json:"LastRunAt,omitempty"`
	LastRunStatus         *string                `json:"LastRunStatus,omitempty"`
	ScheduleCreatedAt     *time.Time             `json:"ScheduleCreatedAt,omitempty"`
	ScheduleLastModified  *time.Time             `json:"ScheduleLastModifiedAt,omitempty"`
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
	WorkerID                 string `json:"workerId"`
	ActiveOrchestrationsCount int   `json:"activeOrchestrationsCount"`
	MaxOrchestrationsCount    int   `json:"maxOrchestrationsCount"`
	ActiveActivitiesCount     int   `json:"activeActivitiesCount"`
	MaxActivitiesCount        int   `json:"maxActivitiesCount"`
	ActiveEntitiesCount       int   `json:"activeEntitiesCount"`
	MaxEntitiesCount          int   `json:"maxEntitiesCount"`
}

// WorkersResult is the response from listing workers.
type WorkersResult struct {
	Workers []Worker `json:"workers"`
}

// --- Agent Models ---

// AgentEntity represents an agent session entity.
type AgentEntity struct {
	Name             string    `json:"name"`
	SessionID        string    `json:"sessionId"`
	EntityID         string    `json:"entityId"`
	LastModified     time.Time `json:"lastModifiedTime"`
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
	Role         string              `json:"role"`
	Content      string              `json:"content,omitempty"`
	FunctionCall *AgentFunctionCall  `json:"functionCall,omitempty"`
	Timestamp    *time.Time          `json:"timestamp,omitempty"`
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
