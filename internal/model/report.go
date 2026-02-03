// Package model defines the data models for the application.
package model

import (
	"time"

	"gorm.io/gorm"
)

// ReportStatus represents the status of a report generation
type ReportStatus string

const (
	ReportStatusPending    ReportStatus = "pending"    // 等待开始
	ReportStatusAnalyzing  ReportStatus = "analyzing"  // 阶段1: 结构分析中
	ReportStatusGenerating ReportStatus = "generating" // 阶段2: 内容生成中
	ReportStatusCompleted  ReportStatus = "completed"  // 完成
	ReportStatusFailed     ReportStatus = "failed"     // 失败
	ReportStatusCancelled  ReportStatus = "cancelled"  // 已取消
)

// SectionStatus represents the status of a report section
type SectionStatus string

const (
	SectionStatusPending   SectionStatus = "pending"
	SectionStatusRunning   SectionStatus = "running"
	SectionStatusCompleted SectionStatus = "completed"
	SectionStatusFailed    SectionStatus = "failed"
	SectionStatusSkipped   SectionStatus = "skipped"
)

// Report represents a code analysis report for a repository
type Report struct {
	ID        string         `gorm:"primarykey;size:20" json:"id"` // xid
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`

	// Repository information
	RepoURL  string `gorm:"size:512;not null;index" json:"repo_url"` // full repository URL
	Ref      string `gorm:"size:255;not null" json:"ref"`            // branch, tag, or commit
	RepoPath string `gorm:"size:1024" json:"repo_path"`              // local workspace path

	// Report type and title
	ReportType string `gorm:"size:50;not null;index" json:"report_type"` // wiki, security, architecture, etc.
	Title      string `gorm:"size:512" json:"title"`                     // report title

	// Status and progress
	Status ReportStatus `gorm:"size:50;not null;default:pending;index" json:"status"`

	// Phase 1 result: report structure outline (JSON)
	// Structure: { sections: [{id, title, description, related_files}] }
	Structure JSONMap `gorm:"type:json" json:"structure,omitempty"`

	// Phase 2 progress
	TotalSections  int `gorm:"default:0" json:"total_sections"`  // total number of sections
	CurrentSection int `gorm:"default:0" json:"current_section"` // current section index (0-based)

	// Final content
	Content string `gorm:"type:text" json:"content,omitempty"` // complete Markdown content

	// Report summary (Phase 3 output)
	// Markdown format, used for report introduction and abstract image generation
	Summary string `gorm:"type:text" json:"summary,omitempty"`

	// Agent configuration
	Agent string `gorm:"size:100" json:"agent,omitempty"` // AI agent used

	// Timing
	StartedAt   *time.Time `json:"started_at,omitempty"`
	CompletedAt *time.Time `json:"completed_at,omitempty"`
	Duration    int64      `json:"duration,omitempty"` // milliseconds

	// Error handling
	ErrorMessage string `gorm:"type:text" json:"error_message,omitempty"`

	// Retry
	RetryCount int `gorm:"default:0" json:"retry_count"` // number of retry attempts

	// Relations
	Sections []ReportSection `gorm:"foreignKey:ReportID" json:"sections,omitempty"`
}

// ReportSection represents a section/chapter in a report
type ReportSection struct {
	ID        uint           `gorm:"primarykey" json:"id"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`

	// Association
	ReportID     string `gorm:"size:20;not null;index" json:"report_id"` // xid reference
	SectionIndex int    `gorm:"not null" json:"section_index"`           // index in sections list (0-based)
	SectionID    string `gorm:"size:100;not null" json:"section_id"`     // section identifier (e.g., "architecture", "security")

	// Hierarchy support (for two-level structure)
	ParentSectionID *string `gorm:"size:100;index" json:"parent_section_id,omitempty"` // parent section ID (for subsections)
	IsLeaf          bool    `gorm:"default:true" json:"is_leaf"`                       // true if this section has no subsections

	// Section metadata
	Title       string `gorm:"size:512" json:"title"`                  // section title
	Description string `gorm:"type:text" json:"description,omitempty"` // section description (for prompt)
	Content     string `gorm:"type:text" json:"content,omitempty"`     // section Markdown content

	// Section summary (generated alongside content in Phase 2)
	// Short summary used as input for Phase 3 report summary generation
	Summary string `gorm:"type:text" json:"summary,omitempty"`

	// Execution status
	Status SectionStatus `gorm:"size:50;not null;default:pending;index" json:"status"`

	// Timing
	StartedAt   *time.Time `json:"started_at,omitempty"`
	CompletedAt *time.Time `json:"completed_at,omitempty"`
	Duration    int64      `json:"duration,omitempty"` // milliseconds

	// Error handling
	ErrorMessage string `gorm:"type:text" json:"error_message,omitempty"`

	// Retry
	RetryCount int `gorm:"default:0" json:"retry_count"` // number of retry attempts

	// Relations
	// Note: Not using OnDelete:CASCADE to avoid SQLite migration issues
	Report Report `json:"-"`
}

// ReportAllModels returns all report-related models for auto-migration
func ReportAllModels() []interface{} {
	return []interface{}{
		&Report{},
		&ReportSection{},
	}
}

