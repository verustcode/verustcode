// Package model defines the data models for the application.
// All models use GORM for ORM operations with SQLite database.
package model

import (
	"database/sql/driver"
	"encoding/json"
	"time"

	"gorm.io/gorm"
)

// StringArray is a custom type for storing string arrays in SQLite
type StringArray []string

// Value implements driver.Valuer interface
func (s StringArray) Value() (driver.Value, error) {
	if len(s) == 0 {
		return "[]", nil
	}
	data, err := json.Marshal(s)
	return string(data), err
}

// Scan implements sql.Scanner interface
func (s *StringArray) Scan(value interface{}) error {
	if value == nil {
		*s = []string{}
		return nil
	}
	var bytes []byte
	switch v := value.(type) {
	case []byte:
		bytes = v
	case string:
		bytes = []byte(v)
	}
	return json.Unmarshal(bytes, s)
}

// JSONMap is a custom type for storing JSON maps in SQLite
type JSONMap map[string]interface{}

// Value implements driver.Valuer interface
func (j JSONMap) Value() (driver.Value, error) {
	if j == nil {
		return "{}", nil
	}
	data, err := json.Marshal(j)
	return string(data), err
}

// Scan implements sql.Scanner interface
func (j *JSONMap) Scan(value interface{}) error {
	if value == nil {
		*j = make(map[string]interface{})
		return nil
	}
	var bytes []byte
	switch v := value.(type) {
	case []byte:
		bytes = v
	case string:
		bytes = []byte(v)
	}
	return json.Unmarshal(bytes, j)
}

// ReviewStatus represents the status of a review
type ReviewStatus string

const (
	ReviewStatusPending   ReviewStatus = "pending"
	ReviewStatusRunning   ReviewStatus = "running"
	ReviewStatusCompleted ReviewStatus = "completed"
	ReviewStatusFailed    ReviewStatus = "failed"
	ReviewStatusCancelled ReviewStatus = "cancelled"
)

// Review represents a code review task
type Review struct {
	ID        string         `gorm:"primarykey;size:20" json:"id"` // xid
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`

	// Review identification
	Ref       string `gorm:"size:255;not null" json:"ref"`                                             // branch, tag, or commit
	CommitSHA string `gorm:"size:64;index;uniqueIndex:idx_pr_url_commit,priority:2" json:"commit_sha"` // commit hash
	PRNumber  int    `gorm:"index" json:"pr_number,omitempty"`                                         // PR/MR number if applicable

	// PR information
	PRURL string `gorm:"size:512;uniqueIndex:idx_pr_url_commit,priority:1" json:"pr_url,omitempty"` // PR/MR URL (unique with commit SHA)

	// Repository information (no longer linked to repositories table)
	RepoURL  string `gorm:"size:512;not null;index" json:"repo_url"` // full repository URL
	RepoPath string `gorm:"size:1024" json:"repo_path"`              // local workspace path

	// Source information
	Source      string `gorm:"size:50;not null;default:cli" json:"source"` // "cli" or "webhook"
	TriggeredBy string `gorm:"size:255" json:"triggered_by,omitempty"`     // trigger source (for webhook)

	// Status and progress
	Status           ReviewStatus `gorm:"size:50;not null;default:pending;index" json:"status"`
	CurrentRuleIndex int          `gorm:"default:0" json:"current_rule_index"`   // current rule index (0-based)
	RetryCount       int          `gorm:"default:0;not null" json:"retry_count"` // number of retry attempts

	// Timing
	StartedAt   *time.Time `json:"started_at,omitempty"`
	CompletedAt *time.Time `json:"completed_at,omitempty"`
	Duration    int64      `json:"duration,omitempty"` // milliseconds

	// PR/Branch metadata
	BranchCreatedAt *time.Time `json:"branch_created_at,omitempty"` // first commit time in branch
	MergedAt        *time.Time `json:"merged_at,omitempty"`         // MR merged/closed time
	Author          string     `gorm:"size:255" json:"author,omitempty"`

	// MR statistics
	RevisionCount int `gorm:"default:1" json:"revision_count"` // MR revision count (opened=1, each sync +1)
	CommitCount   int `gorm:"default:0" json:"commit_count"`   // number of commits in MR

	// Diff statistics
	LinesAdded   int `gorm:"default:0" json:"lines_added"`
	LinesDeleted int `gorm:"default:0" json:"lines_deleted"`
	FilesChanged int `gorm:"default:0" json:"files_changed"`

	// Error handling
	ErrorMessage string `gorm:"type:text" json:"error_message,omitempty"`

	// Relations
	Rules []ReviewRule `gorm:"foreignKey:ReviewID" json:"rules,omitempty"`
}

// RuleStatus represents the status of a review rule
type RuleStatus string

const (
	RuleStatusPending   RuleStatus = "pending"
	RuleStatusRunning   RuleStatus = "running"
	RuleStatusCompleted RuleStatus = "completed"
	RuleStatusFailed    RuleStatus = "failed"
	RuleStatusSkipped   RuleStatus = "skipped"
)

// ReviewRule represents a single review rule execution
type ReviewRule struct {
	ID        uint           `gorm:"primarykey" json:"id"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`

	// Association
	ReviewID  string `gorm:"size:20;not null;index" json:"review_id"` // xid reference
	RuleIndex int    `gorm:"not null" json:"rule_index"`              // index in rules list (0-based)
	RuleID    string `gorm:"size:255;not null;index" json:"rule_id"`  // rule.id from DSL

	// Rule configuration snapshot (JSON)
	RuleConfig JSONMap `gorm:"type:json" json:"rule_config,omitempty"` // ReviewRuleConfig JSON

	// Execution status
	Status RuleStatus `gorm:"size:50;not null;default:pending;index" json:"status"`

	// Multi-run configuration
	MultiRunEnabled bool `gorm:"default:false" json:"multi_run_enabled"`
	MultiRunRuns    int  `gorm:"default:1" json:"multi_run_runs"`    // total number of runs
	CurrentRunIndex int  `gorm:"default:0" json:"current_run_index"` // current run index (0-based)

	// Results
	FindingsCount int `gorm:"default:0" json:"findings_count"` // number of findings

	// Prompt stores the rendered prompt text used for execution
	Prompt string `gorm:"type:text" json:"prompt,omitempty"`

	// Timing
	StartedAt   *time.Time `json:"started_at,omitempty"`
	CompletedAt *time.Time `json:"completed_at,omitempty"`
	Duration    int64      `json:"duration,omitempty"` // milliseconds

	// Error handling
	ErrorMessage string `gorm:"type:text" json:"error_message,omitempty"`

	// Retry
	RetryCount int `gorm:"default:0" json:"retry_count"` // number of retry attempts for this rule

	// Relations
	// Note: Not using OnDelete:CASCADE to avoid SQLite migration issues
	// Soft delete is used for all records, physical deletion should be handled explicitly
	Review  Review          `json:"-"`
	Runs    []ReviewRuleRun `gorm:"foreignKey:ReviewRuleID" json:"runs,omitempty"`
	Results []ReviewResult  `gorm:"foreignKey:ReviewRuleID" json:"results,omitempty"`
}

// RunStatus represents the status of a review rule run
type RunStatus string

const (
	RunStatusPending   RunStatus = "pending"
	RunStatusRunning   RunStatus = "running"
	RunStatusCompleted RunStatus = "completed"
	RunStatusFailed    RunStatus = "failed"
)

// ReviewRuleRun represents a single run in multi-run execution
type ReviewRuleRun struct {
	ID        uint           `gorm:"primarykey" json:"id"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`

	// Association
	ReviewRuleID uint `gorm:"not null;index" json:"review_rule_id"`
	RunIndex     int  `gorm:"not null" json:"run_index"` // run number (0-based)

	// Execution configuration
	Model string `gorm:"size:255" json:"model,omitempty"` // model name used
	Agent string `gorm:"size:100;not null" json:"agent"`  // agent used

	// Execution status
	Status RunStatus `gorm:"size:50;not null;default:pending;index" json:"status"`

	// Results
	FindingsCount int `gorm:"default:0" json:"findings_count"` // number of findings

	// Timing
	StartedAt   *time.Time `json:"started_at,omitempty"`
	CompletedAt *time.Time `json:"completed_at,omitempty"`
	Duration    int64      `json:"duration,omitempty"` // milliseconds

	// Error handling
	ErrorMessage string `gorm:"type:text" json:"error_message,omitempty"`

	// Relations
	// Note: Not using OnDelete:CASCADE to avoid SQLite migration issues
	ReviewRule ReviewRule `json:"-"`
}

// ReviewResult stores the complete AI response as JSON
// The structure is entirely determined by the JSON Schema - no assumptions about fields
type ReviewResult struct {
	ID        uint           `gorm:"primarykey" json:"id"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`

	// Association with ReviewRule
	ReviewRuleID uint `gorm:"not null;index" json:"review_rule_id"`

	// Data stores the complete AI response JSON
	// Structure is defined by JSON Schema, not hardcoded
	Data JSONMap `gorm:"type:json;not null" json:"data"`

	// Relations
	// Note: Not using OnDelete:CASCADE to avoid SQLite migration issues
	ReviewRule ReviewRule `json:"-"`
}

// WebhookStatus represents the status of a webhook delivery
type WebhookStatus string

const (
	WebhookStatusPending WebhookStatus = "pending"
	WebhookStatusSuccess WebhookStatus = "success"
	WebhookStatusFailed  WebhookStatus = "failed"
)

// ReviewResultWebhookLog stores webhook delivery records for review results
// This table tracks all webhook push attempts, enabling debugging and manual retry
type ReviewResultWebhookLog struct {
	ID        uint           `gorm:"primarykey" json:"id"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`

	// RuleID is the review rule identifier
	RuleID string `gorm:"size:255;not null;index" json:"rule_id"`

	// WebhookURL is the target webhook endpoint
	WebhookURL string `gorm:"size:1024;not null" json:"webhook_url"`

	// RequestBody stores the complete request payload (for debugging and manual retry)
	RequestBody string `gorm:"type:text;not null" json:"request_body"`

	// Status indicates the delivery status: pending, success, failed
	Status WebhookStatus `gorm:"size:50;not null;default:pending;index" json:"status"`

	// RetryCount is the number of retry attempts made
	RetryCount int `gorm:"default:0" json:"retry_count"`

	// LastError stores the last error message if delivery failed
	LastError string `gorm:"type:text" json:"last_error,omitempty"`
}

// RepositoryReviewConfig stores repository-level review configuration
// This allows different repositories to use different review rule files
type RepositoryReviewConfig struct {
	ID        uint           `gorm:"primarykey" json:"id"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`

	// Repository identification (use full URL as unique key)
	RepoURL string `gorm:"size:512;not null;uniqueIndex" json:"repo_url"` // full repository URL

	// Review configuration
	ReviewFile string `gorm:"size:255" json:"review_file"` // associated review file name, e.g. "frontend.yaml"

	// Metadata
	Description string `gorm:"size:1024" json:"description,omitempty"` // optional description
}

// AllModels returns all models for auto-migration
func AllModels() []interface{} {
	models := []interface{}{
		&Review{},
		&ReviewRule{},
		&ReviewRuleRun{},
		&ReviewResult{},
		&ReviewResultWebhookLog{},
		&RepositoryReviewConfig{},
	}
	// Add report models
	models = append(models, ReportAllModels()...)
	// Add settings models
	models = append(models, SettingsAllModels()...)
	return models
}

// Statistics models for repository-level analytics

// WeeklyDeliveryTime represents MR delivery time statistics for a week
type WeeklyDeliveryTime struct {
	Week string  `json:"week"` // ISO week format: "2024-W01"
	P50  float64 `json:"p50"`  // 50th percentile (median) in hours
	P90  float64 `json:"p90"`  // 90th percentile in hours
	P95  float64 `json:"p95"`  // 95th percentile in hours
}

// WeeklyCodeChange represents code change statistics for a week
type WeeklyCodeChange struct {
	Week         string `json:"week"`          // ISO week format: "2024-W01"
	LinesAdded   int    `json:"lines_added"`   // total lines added
	LinesDeleted int    `json:"lines_deleted"` // total lines deleted
	NetChange    int    `json:"net_change"`    // net change (added - deleted)
}

// WeeklyFileChange represents file change statistics for a week
type WeeklyFileChange struct {
	Week         string `json:"week"`          // ISO week format: "2024-W01"
	FilesChanged int    `json:"files_changed"` // total files changed
}

// WeeklyMRCount represents MR count statistics for a week
type WeeklyMRCount struct {
	Week  string `json:"week"`  // ISO week format: "2024-W01"
	Count int    `json:"count"` // number of MRs
}

// WeeklyRevision represents MR revision statistics for a week
type WeeklyRevision struct {
	Week           string  `json:"week"`            // ISO week format: "2024-W01"
	AvgRevisions   float64 `json:"avg_revisions"`   // average number of revisions per MR
	TotalRevisions int     `json:"total_revisions"` // total revisions
	MRCount        int     `json:"mr_count"`        // number of MRs in this week

	// Layer statistics (generic design for revision count distribution)
	Layer1Count int `json:"layer_1_count"` // count of MRs in layer 1 (e.g., 1-2 revisions)
	Layer2Count int `json:"layer_2_count"` // count of MRs in layer 2 (e.g., 3-4 revisions)
	Layer3Count int `json:"layer_3_count"` // count of MRs in layer 3 (e.g., 5+ revisions)

	// Layer labels for frontend rendering
	Layer1Label string `json:"layer_1_label"` // label for layer 1 (e.g., "1-2次")
	Layer2Label string `json:"layer_2_label"` // label for layer 2 (e.g., "3-4次")
	Layer3Label string `json:"layer_3_label"` // label for layer 3 (e.g., "5+次")
}

// IssueSeverityStats represents issue count by severity level
type IssueSeverityStats struct {
	Severity string `json:"severity"` // critical, high, medium, low, info
	Count    int    `json:"count"`    // number of issues with this severity
}

// IssueCategoryStats represents issue count by category
type IssueCategoryStats struct {
	Category string `json:"category"` // e.g., security, performance, code-quality
	Count    int    `json:"count"`    // number of issues in this category
}

// RepoStats aggregates all repository statistics
type RepoStats struct {
	DeliveryTimeStats  []WeeklyDeliveryTime `json:"delivery_time_stats"`
	CodeChangeStats    []WeeklyCodeChange   `json:"code_change_stats"`
	FileChangeStats    []WeeklyFileChange   `json:"file_change_stats"`
	MRCountStats       []WeeklyMRCount      `json:"mr_count_stats"`
	RevisionStats      []WeeklyRevision     `json:"revision_stats"`
	IssueSeverityStats []IssueSeverityStats `json:"issue_severity_stats"` // issues grouped by severity
	IssueCategoryStats []IssueCategoryStats `json:"issue_category_stats"` // issues grouped by category
}
