package store

import (
	"database/sql/driver"
	"fmt"
	"time"

	"gorm.io/gorm"

	"github.com/verustcode/verustcode/internal/model"
)

// NullTimeString is a custom type for scanning time values from SQLite.
// SQLite stores time as strings, and GORM raw queries need special handling.
type NullTimeString struct {
	Time  time.Time
	Valid bool
}

// Scan implements the sql.Scanner interface for NullTimeString.
func (nt *NullTimeString) Scan(value interface{}) error {
	if value == nil {
		nt.Time, nt.Valid = time.Time{}, false
		return nil
	}

	switch v := value.(type) {
	case time.Time:
		nt.Time, nt.Valid = v, true
		return nil
	case string:
		if v == "" {
			nt.Time, nt.Valid = time.Time{}, false
			return nil
		}
		// Try parsing common time formats
		formats := []string{
			time.RFC3339Nano,
			time.RFC3339,
			"2006-01-02 15:04:05.999999999-07:00",
			"2006-01-02 15:04:05.999999999",
			"2006-01-02 15:04:05",
			"2006-01-02T15:04:05.999999999Z07:00",
		}
		for _, format := range formats {
			if t, err := time.Parse(format, v); err == nil {
				nt.Time, nt.Valid = t, true
				return nil
			}
		}
		// If all formats fail, return invalid but no error
		nt.Time, nt.Valid = time.Time{}, false
		return nil
	case []byte:
		return nt.Scan(string(v))
	default:
		nt.Time, nt.Valid = time.Time{}, false
		return nil
	}
}

// ToTimePtr converts NullTimeString to *time.Time
func (nt NullTimeString) ToTimePtr() *time.Time {
	if !nt.Valid {
		return nil
	}
	return &nt.Time
}

// Value implements the driver.Valuer interface for NullTimeString.
func (nt NullTimeString) Value() (driver.Value, error) {
	if !nt.Valid {
		return nil, nil
	}
	return nt.Time.Format(time.RFC3339Nano), nil
}

// RepositoryWithStats represents a repository config with review statistics.
type RepositoryWithStats struct {
	ID           uint
	RepoURL      string
	ReviewFile   string
	Description  string
	CreatedAt    time.Time
	UpdatedAt    time.Time
	ReviewCount  int64
	LastReviewAt NullTimeString
}

// RepositoryConfigStore defines operations for RepositoryReviewConfig model.
type RepositoryConfigStore interface {
	// CRUD operations
	Create(config *model.RepositoryReviewConfig) error
	GetByID(id uint) (*model.RepositoryReviewConfig, error)
	GetByRepoURL(repoURL string) (*model.RepositoryReviewConfig, error)
	Update(config *model.RepositoryReviewConfig) error
	Save(config *model.RepositoryReviewConfig) error
	Delete(id uint) error
	DeleteByRepoURL(repoURL string) error

	// Query operations
	List(limit, offset int) ([]model.RepositoryReviewConfig, int64, error)
	ListAll() ([]model.RepositoryReviewConfig, error)
	CountAll() (int64, error)

	// ListWithStats returns repository configs with review statistics.
	// Supports search, sorting by repo_url/review_count/last_review_at, and pagination.
	ListWithStats(search, sortBy, sortOrder string, page, pageSize int) ([]RepositoryWithStats, int64, error)

	// EnsureConfig ensures a config record exists for the given repo URL.
	// If not exists, creates a new record with default values.
	// Returns the config and any error encountered.
	EnsureConfig(repoURL string) (*model.RepositoryReviewConfig, error)

	// UpdateReviewFile updates the review file for a repository.
	UpdateReviewFile(repoURL, reviewFile string) error
}

// repositoryConfigStore implements RepositoryConfigStore using GORM.
type repositoryConfigStore struct {
	db *gorm.DB
}

func newRepositoryConfigStore(db *gorm.DB) RepositoryConfigStore {
	return &repositoryConfigStore{db: db}
}

// CRUD implementations

func (s *repositoryConfigStore) Create(config *model.RepositoryReviewConfig) error {
	return s.db.Create(config).Error
}

func (s *repositoryConfigStore) GetByID(id uint) (*model.RepositoryReviewConfig, error) {
	var config model.RepositoryReviewConfig
	err := s.db.First(&config, id).Error
	if err != nil {
		return nil, err
	}
	return &config, nil
}

func (s *repositoryConfigStore) GetByRepoURL(repoURL string) (*model.RepositoryReviewConfig, error) {
	var config model.RepositoryReviewConfig
	err := s.db.Where("repo_url = ?", repoURL).First(&config).Error
	if err != nil {
		return nil, err
	}
	return &config, nil
}

func (s *repositoryConfigStore) Update(config *model.RepositoryReviewConfig) error {
	return s.db.Model(config).Updates(config).Error
}

func (s *repositoryConfigStore) Save(config *model.RepositoryReviewConfig) error {
	return s.db.Save(config).Error
}

func (s *repositoryConfigStore) Delete(id uint) error {
	return s.db.Delete(&model.RepositoryReviewConfig{}, id).Error
}

func (s *repositoryConfigStore) DeleteByRepoURL(repoURL string) error {
	return s.db.Where("repo_url = ?", repoURL).Delete(&model.RepositoryReviewConfig{}).Error
}

// Query operations

func (s *repositoryConfigStore) List(limit, offset int) ([]model.RepositoryReviewConfig, int64, error) {
	var configs []model.RepositoryReviewConfig
	var total int64

	query := s.db.Model(&model.RepositoryReviewConfig{})

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	err := query.Order("created_at DESC").Limit(limit).Offset(offset).Find(&configs).Error
	return configs, total, err
}

func (s *repositoryConfigStore) ListAll() ([]model.RepositoryReviewConfig, error) {
	var configs []model.RepositoryReviewConfig
	err := s.db.Order("created_at DESC").Find(&configs).Error
	return configs, err
}

func (s *repositoryConfigStore) CountAll() (int64, error) {
	var count int64
	err := s.db.Model(&model.RepositoryReviewConfig{}).Count(&count).Error
	return count, err
}

func (s *repositoryConfigStore) ListWithStats(search, sortBy, sortOrder string, page, pageSize int) ([]RepositoryWithStats, int64, error) {
	// Validate sort_by - only allow specific columns
	validSortColumns := map[string]bool{
		"repo_url":       true,
		"review_count":   true,
		"last_review_at": true,
	}
	if !validSortColumns[sortBy] {
		sortBy = "repo_url" // default sort
	}

	// Validate sort_order
	if sortOrder != "desc" {
		sortOrder = "asc" // default order
	}

	// Build the base query with LEFT JOIN to get review stats
	baseQuery := `
		SELECT 
			rrc.id, rrc.repo_url, rrc.review_file, rrc.description, 
			rrc.created_at, rrc.updated_at,
			COALESCE(stats.review_count, 0) as review_count,
			stats.last_review_at
		FROM repository_review_configs rrc
		LEFT JOIN (
			SELECT repo_url, COUNT(*) as review_count, MAX(created_at) as last_review_at
			FROM reviews
			GROUP BY repo_url
		) stats ON rrc.repo_url = stats.repo_url
	`

	// Build WHERE clause
	var whereClause string
	var args []interface{}
	if search != "" {
		whereClause = " WHERE rrc.repo_url LIKE ?"
		args = append(args, "%"+search+"%")
	}

	// Count total (need separate query)
	countQuery := "SELECT COUNT(*) FROM repository_review_configs rrc" + whereClause
	var total int64
	if err := s.db.Raw(countQuery, args...).Scan(&total).Error; err != nil {
		return nil, 0, err
	}

	// Build ORDER BY clause
	var orderClause string
	switch sortBy {
	case "review_count":
		orderClause = fmt.Sprintf(" ORDER BY review_count %s, rrc.repo_url ASC", sortOrder)
	case "last_review_at":
		// For last_review_at, NULL values should be at the end
		if sortOrder == "desc" {
			orderClause = " ORDER BY last_review_at DESC NULLS LAST, rrc.repo_url ASC"
		} else {
			orderClause = " ORDER BY last_review_at ASC NULLS LAST, rrc.repo_url ASC"
		}
	default:
		orderClause = fmt.Sprintf(" ORDER BY rrc.repo_url %s", sortOrder)
	}

	// Build pagination
	paginationClause := fmt.Sprintf(" LIMIT %d OFFSET %d", pageSize, (page-1)*pageSize)

	// Execute full query
	fullQuery := baseQuery + whereClause + orderClause + paginationClause
	var results []RepositoryWithStats
	if err := s.db.Raw(fullQuery, args...).Scan(&results).Error; err != nil {
		return nil, 0, err
	}

	return results, total, nil
}

// EnsureConfig ensures a config record exists for the given repo URL.
func (s *repositoryConfigStore) EnsureConfig(repoURL string) (*model.RepositoryReviewConfig, error) {
	if repoURL == "" {
		return nil, nil
	}

	config := model.RepositoryReviewConfig{
		RepoURL: repoURL,
	}

	// FirstOrCreate: if record exists, does nothing; if not, creates it
	// The unique index on repo_url ensures no duplicates
	result := s.db.Where("repo_url = ?", repoURL).FirstOrCreate(&config)
	if result.Error != nil {
		return nil, result.Error
	}

	return &config, nil
}

// UpdateReviewFile updates the review file for a repository.
func (s *repositoryConfigStore) UpdateReviewFile(repoURL, reviewFile string) error {
	return s.db.Model(&model.RepositoryReviewConfig{}).
		Where("repo_url = ?", repoURL).
		Update("review_file", reviewFile).Error
}
