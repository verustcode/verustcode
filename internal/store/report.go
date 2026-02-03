package store

import (
	"gorm.io/gorm"

	"github.com/verustcode/verustcode/internal/model"
)

// ReportStore defines operations for Report and ReportSection models.
type ReportStore interface {
	// Report CRUD
	Create(report *model.Report) error
	GetByID(id string) (*model.Report, error)
	GetByIDWithSections(id string) (*model.Report, error)
	Update(report *model.Report) error
	Save(report *model.Report) error
	Delete(id string) error

	// Report status updates
	UpdateStatus(id string, status model.ReportStatus) error
	UpdateStatusWithError(id string, status model.ReportStatus, errMsg string) error
	UpdateStructure(id string, structure model.JSONMap, totalSections int) error
	UpdateContent(id string, content string) error
	UpdateSummary(id string, summary string) error
	UpdateProgress(id string, currentSection int) error

	// Report queries
	List(status, reportType string, page, pageSize int) ([]model.Report, int64, error)
	ListByRepository(repoURL string, limit, offset int) ([]model.Report, int64, error)
	ListByStatus(status model.ReportStatus) ([]model.Report, error)
	ListPendingOrProcessing() ([]model.Report, error)
	ListByType(reportType string, limit, offset int) ([]model.Report, int64, error)
	GetLatestByRepoAndType(repoURL, reportType string) (*model.Report, error)
	GetDistinctRepositories() ([]string, error)
	CountAll() (int64, error)

	// Report cancel
	CancelByID(id string) (int64, error)

	// ReportSection operations
	CreateSection(section *model.ReportSection) error
	BatchCreateSections(sections []model.ReportSection) error
	GetSectionByID(id uint) (*model.ReportSection, error)
	GetSectionsByReportID(reportID string) ([]model.ReportSection, error)
	GetLeafSectionsByReportID(reportID string) ([]model.ReportSection, error)
	UpdateSection(section *model.ReportSection) error
	UpdateSectionStatus(id uint, status model.SectionStatus) error
	UpdateSectionContent(id uint, content, summary string) error
	UpdateSectionStatusWithError(id uint, status model.SectionStatus, errMsg string) error
}

// reportStore implements ReportStore using GORM.
type reportStore struct {
	db *gorm.DB
}

func newReportStore(db *gorm.DB) ReportStore {
	return &reportStore{db: db}
}

// Report CRUD implementations

func (s *reportStore) Create(report *model.Report) error {
	return s.db.Create(report).Error
}

func (s *reportStore) GetByID(id string) (*model.Report, error) {
	var report model.Report
	err := s.db.First(&report, "id = ?", id).Error
	if err != nil {
		return nil, err
	}
	return &report, nil
}

func (s *reportStore) Update(report *model.Report) error {
	return s.db.Model(report).Updates(report).Error
}

func (s *reportStore) Save(report *model.Report) error {
	return s.db.Save(report).Error
}

func (s *reportStore) Delete(id string) error {
	return s.db.Delete(&model.Report{}, "id = ?", id).Error
}

// Report status updates

func (s *reportStore) UpdateStatus(id string, status model.ReportStatus) error {
	return s.db.Model(&model.Report{}).Where("id = ?", id).Update("status", status).Error
}

func (s *reportStore) UpdateStatusWithError(id string, status model.ReportStatus, errMsg string) error {
	return s.db.Model(&model.Report{}).Where("id = ?", id).Updates(map[string]interface{}{
		"status":        status,
		"error_message": errMsg,
	}).Error
}

func (s *reportStore) UpdateStructure(id string, structure model.JSONMap, totalSections int) error {
	return s.db.Model(&model.Report{}).Where("id = ?", id).Updates(map[string]interface{}{
		"structure":      structure,
		"total_sections": totalSections,
	}).Error
}

func (s *reportStore) UpdateContent(id string, content string) error {
	return s.db.Model(&model.Report{}).Where("id = ?", id).Update("content", content).Error
}

func (s *reportStore) UpdateSummary(id string, summary string) error {
	return s.db.Model(&model.Report{}).Where("id = ?", id).Update("summary", summary).Error
}

func (s *reportStore) UpdateProgress(id string, currentSection int) error {
	return s.db.Model(&model.Report{}).Where("id = ?", id).Update("current_section", currentSection).Error
}

// Report queries

func (s *reportStore) ListByRepository(repoURL string, limit, offset int) ([]model.Report, int64, error) {
	var reports []model.Report
	var total int64

	query := s.db.Model(&model.Report{}).Where("repo_url = ?", repoURL)

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	err := query.Order("created_at DESC").Limit(limit).Offset(offset).Find(&reports).Error
	return reports, total, err
}

func (s *reportStore) ListByStatus(status model.ReportStatus) ([]model.Report, error) {
	var reports []model.Report
	err := s.db.Where("status = ?", status).Find(&reports).Error
	return reports, err
}

func (s *reportStore) ListPendingOrProcessing() ([]model.Report, error) {
	var reports []model.Report
	err := s.db.Where("status IN ?", []model.ReportStatus{
		model.ReportStatusPending,
		model.ReportStatusAnalyzing,
		model.ReportStatusGenerating,
	}).Order("created_at ASC").Find(&reports).Error
	return reports, err
}

func (s *reportStore) ListByType(reportType string, limit, offset int) ([]model.Report, int64, error) {
	var reports []model.Report
	var total int64

	query := s.db.Model(&model.Report{}).Where("report_type = ?", reportType)

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	err := query.Order("created_at DESC").Limit(limit).Offset(offset).Find(&reports).Error
	return reports, total, err
}

func (s *reportStore) GetLatestByRepoAndType(repoURL, reportType string) (*model.Report, error) {
	var report model.Report
	err := s.db.Where("repo_url = ? AND report_type = ?", repoURL, reportType).
		Order("created_at DESC").
		First(&report).Error
	if err != nil {
		return nil, err
	}
	return &report, nil
}

func (s *reportStore) CountAll() (int64, error) {
	var count int64
	err := s.db.Model(&model.Report{}).Count(&count).Error
	return count, err
}

// GetByIDWithSections retrieves a report by ID and preloads its sections.
func (s *reportStore) GetByIDWithSections(id string) (*model.Report, error) {
	var report model.Report
	err := s.db.Preload("Sections", func(db *gorm.DB) *gorm.DB {
		return db.Order("section_index ASC")
	}).First(&report, "id = ?", id).Error
	if err != nil {
		return nil, err
	}
	return &report, nil
}

// List lists reports with optional filters and pagination.
func (s *reportStore) List(status, reportType string, page, pageSize int) ([]model.Report, int64, error) {
	var reports []model.Report
	var total int64

	query := s.db.Model(&model.Report{})

	if status != "" {
		query = query.Where("status = ?", status)
	}
	if reportType != "" {
		query = query.Where("report_type = ?", reportType)
	}

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * pageSize
	err := query.Order("created_at DESC").Offset(offset).Limit(pageSize).Find(&reports).Error
	return reports, total, err
}

// GetDistinctRepositories returns a list of unique repository URLs from all reports.
func (s *reportStore) GetDistinctRepositories() ([]string, error) {
	var repos []string
	err := s.db.Model(&model.Report{}).Distinct("repo_url").Pluck("repo_url", &repos).Error
	return repos, err
}

// CancelByID cancels a report by ID if it's in pending, analyzing, or generating status.
func (s *reportStore) CancelByID(id string) (int64, error) {
	result := s.db.Model(&model.Report{}).
		Where("id = ?", id).
		Where("status IN ?", []model.ReportStatus{model.ReportStatusPending, model.ReportStatusAnalyzing, model.ReportStatusGenerating}).
		Update("status", model.ReportStatusCancelled)
	return result.RowsAffected, result.Error
}

// ReportSection operations

func (s *reportStore) CreateSection(section *model.ReportSection) error {
	return s.db.Create(section).Error
}

func (s *reportStore) BatchCreateSections(sections []model.ReportSection) error {
	if len(sections) == 0 {
		return nil
	}
	return s.db.Create(&sections).Error
}

func (s *reportStore) GetSectionByID(id uint) (*model.ReportSection, error) {
	var section model.ReportSection
	err := s.db.First(&section, id).Error
	if err != nil {
		return nil, err
	}
	return &section, nil
}

func (s *reportStore) GetSectionsByReportID(reportID string) ([]model.ReportSection, error) {
	var sections []model.ReportSection
	err := s.db.Where("report_id = ?", reportID).Order("section_index ASC").Find(&sections).Error
	return sections, err
}

func (s *reportStore) GetLeafSectionsByReportID(reportID string) ([]model.ReportSection, error) {
	var sections []model.ReportSection
	err := s.db.Where("report_id = ? AND is_leaf = ?", reportID, true).Order("section_index ASC").Find(&sections).Error
	return sections, err
}

func (s *reportStore) UpdateSection(section *model.ReportSection) error {
	return s.db.Save(section).Error
}

func (s *reportStore) UpdateSectionStatus(id uint, status model.SectionStatus) error {
	return s.db.Model(&model.ReportSection{}).Where("id = ?", id).Update("status", status).Error
}

func (s *reportStore) UpdateSectionContent(id uint, content, summary string) error {
	return s.db.Model(&model.ReportSection{}).Where("id = ?", id).Updates(map[string]interface{}{
		"content": content,
		"summary": summary,
	}).Error
}

func (s *reportStore) UpdateSectionStatusWithError(id uint, status model.SectionStatus, errMsg string) error {
	return s.db.Model(&model.ReportSection{}).Where("id = ?", id).Updates(map[string]interface{}{
		"status":        status,
		"error_message": errMsg,
	}).Error
}
