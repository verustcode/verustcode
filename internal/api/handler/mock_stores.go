// Package handler provides mock store implementations for testing.
package handler

import (
	"sync"
	"time"

	"gorm.io/gorm"

	"github.com/verustcode/verustcode/internal/model"
	"github.com/verustcode/verustcode/internal/store"
)

// MockReviewStore provides in-memory implementation of ReviewStore for testing.
type MockReviewStore struct {
	mu      sync.RWMutex
	reviews map[string]*model.Review
	rules   map[uint]*model.ReviewRule
}

// NewMockReviewStore creates a new mock review store.
func NewMockReviewStore() *MockReviewStore {
	return &MockReviewStore{
		reviews: make(map[string]*model.Review),
		rules:   make(map[uint]*model.ReviewRule),
	}
}

func (m *MockReviewStore) Create(review *model.Review) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.reviews[review.ID] = review
	return nil
}

func (m *MockReviewStore) GetByID(id string) (*model.Review, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	review, ok := m.reviews[id]
	if !ok {
		return nil, gorm.ErrRecordNotFound
	}
	return review, nil
}

func (m *MockReviewStore) GetByIDWithDetails(id string) (*model.Review, error) {
	return m.GetByID(id)
}

func (m *MockReviewStore) GetByIDWithRules(id string) (*model.Review, error) {
	return m.GetByID(id)
}

func (m *MockReviewStore) Update(review *model.Review) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.reviews[review.ID]; !ok {
		return gorm.ErrRecordNotFound
	}
	m.reviews[review.ID] = review
	return nil
}

func (m *MockReviewStore) Save(review *model.Review) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.reviews[review.ID] = review
	return nil
}

func (m *MockReviewStore) Delete(id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.reviews, id)
	return nil
}

func (m *MockReviewStore) UpdateStatus(id string, status model.ReviewStatus) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	review, ok := m.reviews[id]
	if !ok {
		return gorm.ErrRecordNotFound
	}
	review.Status = status
	return nil
}

func (m *MockReviewStore) UpdateStatusWithError(id string, status model.ReviewStatus, errMsg string) error {
	return m.UpdateStatus(id, status)
}

func (m *MockReviewStore) UpdateStatusWithErrorAndCompletedAt(id string, status model.ReviewStatus, errMsg string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	review, ok := m.reviews[id]
	if !ok {
		return gorm.ErrRecordNotFound
	}
	review.Status = status
	review.ErrorMessage = errMsg
	now := time.Now()
	review.CompletedAt = &now
	return nil
}

func (m *MockReviewStore) UpdateStatusToRunningIfPending(id string, startedAt time.Time) (bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	review, ok := m.reviews[id]
	if !ok {
		return false, gorm.ErrRecordNotFound
	}
	if review.Status == model.ReviewStatusPending {
		review.Status = model.ReviewStatusRunning
		review.StartedAt = &startedAt
		return true, nil
	}
	return false, nil
}

func (m *MockReviewStore) UpdateStatusIfAllowed(id string, newStatus model.ReviewStatus, allowedStatuses []model.ReviewStatus) (int64, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	review, ok := m.reviews[id]
	if !ok {
		return 0, gorm.ErrRecordNotFound
	}
	for _, allowed := range allowedStatuses {
		if review.Status == allowed {
			review.Status = newStatus
			return 1, nil
		}
	}
	return 0, nil
}

func (m *MockReviewStore) UpdateProgress(id string, currentRuleIndex int) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	review, ok := m.reviews[id]
	if !ok {
		return gorm.ErrRecordNotFound
	}
	review.CurrentRuleIndex = currentRuleIndex
	return nil
}

func (m *MockReviewStore) UpdateCurrentRuleIndex(reviewID string, index int) error {
	return m.UpdateProgress(reviewID, index)
}

func (m *MockReviewStore) UpdateRepoPath(reviewID, repoPath string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	review, ok := m.reviews[reviewID]
	if !ok {
		return gorm.ErrRecordNotFound
	}
	review.RepoPath = repoPath
	return nil
}

func (m *MockReviewStore) UpdateMetadata(reviewID string, updates map[string]interface{}) error {
	return nil
}

func (m *MockReviewStore) IncrementRetryCount(id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	review, ok := m.reviews[id]
	if !ok {
		return gorm.ErrRecordNotFound
	}
	review.RetryCount++
	return nil
}

func (m *MockReviewStore) List(statusFilter string, limit, offset int) ([]model.Review, int64, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	var reviews []model.Review
	for _, review := range m.reviews {
		if statusFilter == "" || string(review.Status) == statusFilter {
			reviews = append(reviews, *review)
		}
	}
	total := int64(len(reviews))
	if offset >= len(reviews) {
		return []model.Review{}, total, nil
	}
	end := offset + limit
	if end > len(reviews) {
		end = len(reviews)
	}
	return reviews[offset:end], total, nil
}

func (m *MockReviewStore) ListByRepository(repoURL string, limit, offset int) ([]model.Review, int64, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	var reviews []model.Review
	for _, review := range m.reviews {
		if review.RepoURL == repoURL {
			reviews = append(reviews, *review)
		}
	}
	total := int64(len(reviews))
	if offset >= len(reviews) {
		return []model.Review{}, total, nil
	}
	end := offset + limit
	if end > len(reviews) {
		end = len(reviews)
	}
	return reviews[offset:end], total, nil
}

func (m *MockReviewStore) ListByStatus(status model.ReviewStatus) ([]model.Review, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	var reviews []model.Review
	for _, review := range m.reviews {
		if review.Status == status {
			reviews = append(reviews, *review)
		}
	}
	return reviews, nil
}

func (m *MockReviewStore) ListPendingOrRunning() ([]model.Review, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	var reviews []model.Review
	for _, review := range m.reviews {
		if review.Status == model.ReviewStatusPending || review.Status == model.ReviewStatusRunning {
			reviews = append(reviews, *review)
		}
	}
	return reviews, nil
}

func (m *MockReviewStore) GetByPRURLAndCommit(prURL, commitSHA string) (*model.Review, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	for _, review := range m.reviews {
		if review.PRURL == prURL && review.CommitSHA == commitSHA {
			return review, nil
		}
	}
	return nil, gorm.ErrRecordNotFound
}

func (m *MockReviewStore) CreateRule(rule *model.ReviewRule) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if rule.ID == 0 {
		rule.ID = uint(len(m.rules) + 1)
	}
	m.rules[rule.ID] = rule
	return nil
}

func (m *MockReviewStore) BatchCreateRules(rules []model.ReviewRule) error {
	for i := range rules {
		if err := m.CreateRule(&rules[i]); err != nil {
			return err
		}
	}
	return nil
}

func (m *MockReviewStore) GetRuleByID(id uint) (*model.ReviewRule, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	rule, ok := m.rules[id]
	if !ok {
		return nil, gorm.ErrRecordNotFound
	}
	return rule, nil
}

func (m *MockReviewStore) GetRulesByReviewID(reviewID string) ([]model.ReviewRule, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	var rules []model.ReviewRule
	for _, rule := range m.rules {
		if rule.ReviewID == reviewID {
			rules = append(rules, *rule)
		}
	}
	return rules, nil
}

func (m *MockReviewStore) UpdateRule(rule *model.ReviewRule) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.rules[rule.ID]; !ok {
		return gorm.ErrRecordNotFound
	}
	m.rules[rule.ID] = rule
	return nil
}

func (m *MockReviewStore) UpdateRuleStatus(id uint, status model.RuleStatus) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	rule, ok := m.rules[id]
	if !ok {
		return gorm.ErrRecordNotFound
	}
	rule.Status = status
	return nil
}

func (m *MockReviewStore) UpdateRuleStatusWithError(id uint, status model.RuleStatus, errMsg string) error {
	return m.UpdateRuleStatus(id, status)
}

func (m *MockReviewStore) CreateRun(run *model.ReviewRuleRun) error {
	return nil
}

func (m *MockReviewStore) GetRunByID(id uint) (*model.ReviewRuleRun, error) {
	return nil, gorm.ErrRecordNotFound
}

func (m *MockReviewStore) GetRunsByRuleID(ruleID uint) ([]model.ReviewRuleRun, error) {
	return nil, nil
}

func (m *MockReviewStore) DeleteReviewRuleRunsByRuleID(ruleID uint) error {
	return nil
}

func (m *MockReviewStore) UpdateRun(run *model.ReviewRuleRun) error {
	return nil
}

func (m *MockReviewStore) UpdateRunStatus(id uint, status model.RunStatus) error {
	return nil
}

func (m *MockReviewStore) CreateResult(result *model.ReviewResult) error {
	return nil
}

func (m *MockReviewStore) DeleteReviewResultsByRuleID(ruleID uint) error {
	return nil
}

func (m *MockReviewStore) GetResultsByRuleID(ruleID uint) ([]model.ReviewResult, error) {
	return nil, nil
}

func (m *MockReviewStore) GetResultsByReviewID(reviewID string) ([]model.ReviewResult, error) {
	return nil, nil
}

func (m *MockReviewStore) CreateWebhookLog(log *model.ReviewResultWebhookLog) error {
	return nil
}

func (m *MockReviewStore) UpdateWebhookLog(log *model.ReviewResultWebhookLog) error {
	return nil
}

func (m *MockReviewStore) GetPendingWebhookLogs() ([]model.ReviewResultWebhookLog, error) {
	return nil, nil
}

func (m *MockReviewStore) CountByStatusAndDateRange(status model.ReviewStatus, start, end time.Time) (int64, error) {
	return 0, nil
}

func (m *MockReviewStore) GetReviewsWithResultsByRepository(repoURL string, limit, offset int) ([]model.Review, error) {
	reviews, _, err := m.ListByRepository(repoURL, limit, offset)
	return reviews, err
}

func (m *MockReviewStore) CountAll() (int64, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return int64(len(m.reviews)), nil
}

func (m *MockReviewStore) CountCreatedAfter(start time.Time) (int64, error) {
	return 0, nil
}

func (m *MockReviewStore) CountByStatusOnly(status model.ReviewStatus) (int64, error) {
	return 0, nil
}

func (m *MockReviewStore) CountByStatusAndCompletedAfter(status model.ReviewStatus, start time.Time) (int64, error) {
	return 0, nil
}

func (m *MockReviewStore) CountCompletedOrFailedAfter(start time.Time) (int64, error) {
	return 0, nil
}

func (m *MockReviewStore) CountCompletedAfter(start time.Time) (int64, error) {
	return 0, nil
}

func (m *MockReviewStore) GetAverageDurationAfter(start time.Time) (float64, error) {
	return 0, nil
}

func (m *MockReviewStore) ListCompletedByRepoAndDateRange(repoURL string, start time.Time) ([]model.Review, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	var reviews []model.Review
	for _, review := range m.reviews {
		// Filter by repo URL and completed status
		if review.RepoURL == repoURL || repoURL == "" {
			if review.Status == model.ReviewStatusCompleted {
				// Filter by date range
				if review.CreatedAt.After(start) || review.CreatedAt.Equal(start) {
					reviews = append(reviews, *review)
				}
			}
		}
	}
	return reviews, nil
}

func (m *MockReviewStore) GetReviewResultsByReviewIDs(reviewIDs []string) ([]model.ReviewResult, error) {
	// Return empty results for now - can be extended if needed
	return []model.ReviewResult{}, nil
}

func (m *MockReviewStore) GetMaxRevisionByPRURL(prURL string) (int, error) {
	return 0, nil
}

func (m *MockReviewStore) UpdateMergedAtByPRURL(prURL string, mergedAt time.Time) (int64, error) {
	return 0, nil
}

func (m *MockReviewStore) FindPreviousReviewResult(prURL, ruleID, currentReviewID string) (string, bool, error) {
	return "", false, nil
}

func (m *MockReviewStore) ResetReviewState(reviewID string, retryCount int) error {
	return nil
}

func (m *MockReviewStore) ResetRuleState(ruleID string, reviewID string, ruleRetryCount, reviewRetryCount int) error {
	return nil
}

func (m *MockReviewStore) GetAllFindingsWithRepoInfo(repoURL string) ([]store.FindingWithRepoInfo, error) {
	// Return empty results for mock
	return []store.FindingWithRepoInfo{}, nil
}

// MockReportStore provides in-memory implementation of ReportStore for testing.
type MockReportStore struct {
	mu      sync.RWMutex
	reports map[string]*model.Report
}

// NewMockReportStore creates a new mock report store.
func NewMockReportStore() *MockReportStore {
	return &MockReportStore{
		reports: make(map[string]*model.Report),
	}
}

func (m *MockReportStore) Create(report *model.Report) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.reports[report.ID] = report
	return nil
}

func (m *MockReportStore) GetByID(id string) (*model.Report, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	report, ok := m.reports[id]
	if !ok {
		return nil, gorm.ErrRecordNotFound
	}
	return report, nil
}

func (m *MockReportStore) GetByIDWithSections(id string) (*model.Report, error) {
	return m.GetByID(id)
}

func (m *MockReportStore) Update(report *model.Report) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.reports[report.ID]; !ok {
		return gorm.ErrRecordNotFound
	}
	m.reports[report.ID] = report
	return nil
}

func (m *MockReportStore) Save(report *model.Report) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.reports[report.ID] = report
	return nil
}

func (m *MockReportStore) Delete(id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.reports, id)
	return nil
}

func (m *MockReportStore) UpdateStatus(id string, status model.ReportStatus) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	report, ok := m.reports[id]
	if !ok {
		return gorm.ErrRecordNotFound
	}
	report.Status = status
	return nil
}

func (m *MockReportStore) UpdateStatusWithError(id string, status model.ReportStatus, errMsg string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	report, ok := m.reports[id]
	if !ok {
		return gorm.ErrRecordNotFound
	}
	report.Status = status
	report.ErrorMessage = errMsg
	return nil
}

func (m *MockReportStore) UpdateStructure(id string, structure model.JSONMap, totalSections int) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	report, ok := m.reports[id]
	if !ok {
		return gorm.ErrRecordNotFound
	}
	report.Structure = structure
	report.TotalSections = totalSections
	return nil
}

func (m *MockReportStore) UpdateContent(id string, content string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	report, ok := m.reports[id]
	if !ok {
		return gorm.ErrRecordNotFound
	}
	report.Content = content
	return nil
}

func (m *MockReportStore) UpdateSummary(id string, summary string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	report, ok := m.reports[id]
	if !ok {
		return gorm.ErrRecordNotFound
	}
	report.Summary = summary
	return nil
}

func (m *MockReportStore) UpdateProgress(id string, currentSection int) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	report, ok := m.reports[id]
	if !ok {
		return gorm.ErrRecordNotFound
	}
	report.CurrentSection = currentSection
	return nil
}

func (m *MockReportStore) List(status, reportType string, page, pageSize int) ([]model.Report, int64, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	var reports []model.Report
	for _, report := range m.reports {
		if (status == "" || string(report.Status) == status) &&
			(reportType == "" || report.ReportType == reportType) {
			reports = append(reports, *report)
		}
	}
	total := int64(len(reports))
	offset := (page - 1) * pageSize
	if offset >= len(reports) {
		return []model.Report{}, total, nil
	}
	end := offset + pageSize
	if end > len(reports) {
		end = len(reports)
	}
	return reports[offset:end], total, nil
}

func (m *MockReportStore) ListByRepository(repoURL string, limit, offset int) ([]model.Report, int64, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	var reports []model.Report
	for _, report := range m.reports {
		if report.RepoURL == repoURL {
			reports = append(reports, *report)
		}
	}
	total := int64(len(reports))
	if offset >= len(reports) {
		return []model.Report{}, total, nil
	}
	end := offset + limit
	if end > len(reports) {
		end = len(reports)
	}
	return reports[offset:end], total, nil
}

func (m *MockReportStore) ListByStatus(status model.ReportStatus) ([]model.Report, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	var reports []model.Report
	for _, report := range m.reports {
		if report.Status == status {
			reports = append(reports, *report)
		}
	}
	return reports, nil
}

func (m *MockReportStore) ListPendingOrProcessing() ([]model.Report, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	var reports []model.Report
	for _, report := range m.reports {
		if report.Status == model.ReportStatusPending ||
			report.Status == model.ReportStatusAnalyzing ||
			report.Status == model.ReportStatusGenerating {
			reports = append(reports, *report)
		}
	}
	return reports, nil
}

func (m *MockReportStore) ListByType(reportType string, limit, offset int) ([]model.Report, int64, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	var reports []model.Report
	for _, report := range m.reports {
		if report.ReportType == reportType {
			reports = append(reports, *report)
		}
	}
	total := int64(len(reports))
	if offset >= len(reports) {
		return []model.Report{}, total, nil
	}
	end := offset + limit
	if end > len(reports) {
		end = len(reports)
	}
	return reports[offset:end], total, nil
}

func (m *MockReportStore) GetLatestByRepoAndType(repoURL, reportType string) (*model.Report, error) {
	return nil, gorm.ErrRecordNotFound
}

func (m *MockReportStore) GetDistinctRepositories() ([]string, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	repoMap := make(map[string]bool)
	for _, report := range m.reports {
		repoMap[report.RepoURL] = true
	}
	var repos []string
	for repo := range repoMap {
		repos = append(repos, repo)
	}
	return repos, nil
}

func (m *MockReportStore) CountAll() (int64, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return int64(len(m.reports)), nil
}

func (m *MockReportStore) CancelByID(id string) (int64, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	report, ok := m.reports[id]
	if !ok {
		return 0, gorm.ErrRecordNotFound
	}
	report.Status = model.ReportStatusCancelled
	return 1, nil
}

func (m *MockReportStore) CreateSection(section *model.ReportSection) error {
	return nil
}

func (m *MockReportStore) BatchCreateSections(sections []model.ReportSection) error {
	return nil
}

func (m *MockReportStore) GetSectionByID(id uint) (*model.ReportSection, error) {
	return nil, gorm.ErrRecordNotFound
}

func (m *MockReportStore) GetSectionsByReportID(reportID string) ([]model.ReportSection, error) {
	return nil, nil
}

func (m *MockReportStore) GetLeafSectionsByReportID(reportID string) ([]model.ReportSection, error) {
	return nil, nil
}

func (m *MockReportStore) UpdateSection(section *model.ReportSection) error {
	return nil
}

func (m *MockReportStore) UpdateSectionStatus(id uint, status model.SectionStatus) error {
	return nil
}

func (m *MockReportStore) UpdateSectionContent(id uint, content, summary string) error {
	return nil
}

func (m *MockReportStore) UpdateSectionStatusWithError(id uint, status model.SectionStatus, errMsg string) error {
	return nil
}

// MockSettingsStore provides in-memory implementation of SettingsStore for testing.
type MockSettingsStore struct {
	mu       sync.RWMutex
	settings map[string]*model.SystemSetting
}

// NewMockSettingsStore creates a new mock settings store.
func NewMockSettingsStore() *MockSettingsStore {
	return &MockSettingsStore{
		settings: make(map[string]*model.SystemSetting),
	}
}

func (m *MockSettingsStore) Get(category, key string) (*model.SystemSetting, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	setting, ok := m.settings[category+":"+key]
	if !ok {
		return nil, gorm.ErrRecordNotFound
	}
	return setting, nil
}

func (m *MockSettingsStore) GetByCategory(category string) ([]model.SystemSetting, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	var settings []model.SystemSetting
	for _, setting := range m.settings {
		if setting.Category == category {
			settings = append(settings, *setting)
		}
	}
	return settings, nil
}

func (m *MockSettingsStore) GetAll() ([]model.SystemSetting, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	var settings []model.SystemSetting
	for _, setting := range m.settings {
		settings = append(settings, *setting)
	}
	return settings, nil
}

func (m *MockSettingsStore) Create(setting *model.SystemSetting) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.settings[setting.Category+":"+setting.Key] = setting
	return nil
}

func (m *MockSettingsStore) Update(setting *model.SystemSetting) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.settings[setting.Category+":"+setting.Key]; !ok {
		return gorm.ErrRecordNotFound
	}
	m.settings[setting.Category+":"+setting.Key] = setting
	return nil
}

func (m *MockSettingsStore) Save(setting *model.SystemSetting) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.settings[setting.Category+":"+setting.Key] = setting
	return nil
}

func (m *MockSettingsStore) Delete(category, key string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.settings, category+":"+key)
	return nil
}

func (m *MockSettingsStore) DeleteByCategory(category string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	for k := range m.settings {
		if m.settings[k].Category == category {
			delete(m.settings, k)
		}
	}
	return nil
}

func (m *MockSettingsStore) DeleteSetting(setting *model.SystemSetting) error {
	return m.Delete(setting.Category, setting.Key)
}

func (m *MockSettingsStore) Count() (int64, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return int64(len(m.settings)), nil
}

func (m *MockSettingsStore) BatchUpsert(settings []model.SystemSetting) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	for i := range settings {
		m.settings[settings[i].Category+":"+settings[i].Key] = &settings[i]
	}
	return nil
}

func (m *MockSettingsStore) WithTx(tx *gorm.DB) store.SettingsStore {
	return m
}

// MockRepositoryConfigStore provides in-memory implementation of RepositoryConfigStore for testing.
type MockRepositoryConfigStore struct {
	mu      sync.RWMutex
	configs map[uint]*model.RepositoryReviewConfig
}

// NewMockRepositoryConfigStore creates a new mock repository config store.
func NewMockRepositoryConfigStore() *MockRepositoryConfigStore {
	return &MockRepositoryConfigStore{
		configs: make(map[uint]*model.RepositoryReviewConfig),
	}
}

func (m *MockRepositoryConfigStore) Create(config *model.RepositoryReviewConfig) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if config.ID == 0 {
		config.ID = uint(len(m.configs) + 1)
	}
	m.configs[config.ID] = config
	return nil
}

func (m *MockRepositoryConfigStore) GetByID(id uint) (*model.RepositoryReviewConfig, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	config, ok := m.configs[id]
	if !ok {
		return nil, gorm.ErrRecordNotFound
	}
	return config, nil
}

func (m *MockRepositoryConfigStore) GetByRepoURL(repoURL string) (*model.RepositoryReviewConfig, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	for _, config := range m.configs {
		if config.RepoURL == repoURL {
			return config, nil
		}
	}
	return nil, gorm.ErrRecordNotFound
}

func (m *MockRepositoryConfigStore) Update(config *model.RepositoryReviewConfig) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.configs[config.ID]; !ok {
		return gorm.ErrRecordNotFound
	}
	m.configs[config.ID] = config
	return nil
}

func (m *MockRepositoryConfigStore) Delete(id uint) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.configs, id)
	return nil
}

func (m *MockRepositoryConfigStore) List(limit, offset int) ([]model.RepositoryReviewConfig, int64, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	var configs []model.RepositoryReviewConfig
	for _, config := range m.configs {
		configs = append(configs, *config)
	}
	total := int64(len(configs))
	if offset >= len(configs) {
		return []model.RepositoryReviewConfig{}, total, nil
	}
	end := offset + limit
	if end > len(configs) {
		end = len(configs)
	}
	return configs[offset:end], total, nil
}

func (m *MockRepositoryConfigStore) Count() (int64, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return int64(len(m.configs)), nil
}

func (m *MockRepositoryConfigStore) Save(config *model.RepositoryReviewConfig) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.configs[config.ID] = config
	return nil
}

func (m *MockRepositoryConfigStore) DeleteByRepoURL(repoURL string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	for id, config := range m.configs {
		if config.RepoURL == repoURL {
			delete(m.configs, id)
			return nil
		}
	}
	return gorm.ErrRecordNotFound
}

func (m *MockRepositoryConfigStore) ListAll() ([]model.RepositoryReviewConfig, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	var configs []model.RepositoryReviewConfig
	for _, config := range m.configs {
		configs = append(configs, *config)
	}
	return configs, nil
}

func (m *MockRepositoryConfigStore) CountAll() (int64, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return int64(len(m.configs)), nil
}

func (m *MockRepositoryConfigStore) ListWithStats(search, sortBy, sortOrder string, page, pageSize int) ([]store.RepositoryWithStats, int64, error) {
	return nil, 0, nil
}

func (m *MockRepositoryConfigStore) EnsureConfig(repoURL string) (*model.RepositoryReviewConfig, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, config := range m.configs {
		if config.RepoURL == repoURL {
			return config, nil
		}
	}
	config := &model.RepositoryReviewConfig{
		ID:      uint(len(m.configs) + 1),
		RepoURL: repoURL,
	}
	m.configs[config.ID] = config
	return config, nil
}

func (m *MockRepositoryConfigStore) UpdateReviewFile(repoURL, reviewFile string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, config := range m.configs {
		if config.RepoURL == repoURL {
			config.ReviewFile = reviewFile
			return nil
		}
	}
	return gorm.ErrRecordNotFound
}
