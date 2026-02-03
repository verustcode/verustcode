// Package model defines the data models for the application.
// This file defines system settings models for runtime configuration storage.
package model

import (
	"time"

	"gorm.io/gorm"
)

// SettingCategory represents the category of a system setting
type SettingCategory string

const (
	SettingCategoryApp           SettingCategory = "app"           // Application settings (subtitle, etc.)
	SettingCategoryGit           SettingCategory = "git"           // Git providers configuration
	SettingCategoryAgents        SettingCategory = "agents"        // AI agents configuration
	SettingCategoryReview        SettingCategory = "review"        // Review process settings
	SettingCategoryReport        SettingCategory = "report"        // Report generation settings
	SettingCategoryNotifications SettingCategory = "notifications" // Notification settings
)

// AllSettingCategories returns all valid setting categories
func AllSettingCategories() []SettingCategory {
	return []SettingCategory{
		SettingCategoryApp,
		SettingCategoryGit,
		SettingCategoryAgents,
		SettingCategoryReview,
		SettingCategoryReport,
		SettingCategoryNotifications,
	}
}

// SettingValueType represents the type of a setting value
type SettingValueType string

const (
	SettingValueTypeString  SettingValueType = "string"
	SettingValueTypeNumber  SettingValueType = "number"
	SettingValueTypeBoolean SettingValueType = "boolean"
	SettingValueTypeArray   SettingValueType = "array"
	SettingValueTypeObject  SettingValueType = "object"
)

// SystemSetting stores a single system configuration item.
// Configuration is stored in category + key format for flexible querying.
type SystemSetting struct {
	ID        uint           `gorm:"primarykey" json:"id"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`

	// Category groups related settings together
	// Valid values: app, git, agents, review, report, notifications
	Category string `gorm:"size:50;not null;index;uniqueIndex:idx_category_key,priority:1" json:"category"`

	// Key is the unique identifier within a category
	Key string `gorm:"size:100;not null;uniqueIndex:idx_category_key,priority:2" json:"key"`

	// Value stores the setting value as JSON string
	Value string `gorm:"type:text;not null" json:"value"`

	// ValueType indicates the type of the value for proper parsing
	ValueType string `gorm:"size:20;not null;default:string" json:"value_type"`
}

// SettingsAllModels returns all settings-related models for auto-migration
func SettingsAllModels() []interface{} {
	return []interface{}{
		&SystemSetting{},
	}
}
