package store

import (
	"gorm.io/gorm"

	"github.com/verustcode/verustcode/internal/model"
)

// SettingsStore defines operations for SystemSetting models.
type SettingsStore interface {
	// SystemSetting CRUD
	Get(category, key string) (*model.SystemSetting, error)
	GetByCategory(category string) ([]model.SystemSetting, error)
	GetAll() ([]model.SystemSetting, error)
	Create(setting *model.SystemSetting) error
	Update(setting *model.SystemSetting) error
	Save(setting *model.SystemSetting) error
	Delete(category, key string) error
	DeleteByCategory(category string) error
	DeleteSetting(setting *model.SystemSetting) error

	// Query operations
	Count() (int64, error)

	// Batch operations
	BatchUpsert(settings []model.SystemSetting) error

	// Transaction support
	WithTx(tx *gorm.DB) SettingsStore
}

// settingsStore implements SettingsStore using GORM.
type settingsStore struct {
	db *gorm.DB
}

func newSettingsStore(db *gorm.DB) SettingsStore {
	return &settingsStore{db: db}
}

// SystemSetting CRUD implementations

func (s *settingsStore) Get(category, key string) (*model.SystemSetting, error) {
	var setting model.SystemSetting
	err := s.db.Where("category = ? AND key = ?", category, key).First(&setting).Error
	if err != nil {
		return nil, err
	}
	return &setting, nil
}

func (s *settingsStore) GetByCategory(category string) ([]model.SystemSetting, error) {
	var settings []model.SystemSetting
	err := s.db.Where("category = ?", category).Find(&settings).Error
	return settings, err
}

func (s *settingsStore) GetAll() ([]model.SystemSetting, error) {
	var settings []model.SystemSetting
	err := s.db.Find(&settings).Error
	return settings, err
}

func (s *settingsStore) Create(setting *model.SystemSetting) error {
	return s.db.Create(setting).Error
}

func (s *settingsStore) Update(setting *model.SystemSetting) error {
	return s.db.Model(setting).Updates(setting).Error
}

func (s *settingsStore) Save(setting *model.SystemSetting) error {
	return s.db.Save(setting).Error
}

func (s *settingsStore) Delete(category, key string) error {
	return s.db.Where("category = ? AND key = ?", category, key).Delete(&model.SystemSetting{}).Error
}

func (s *settingsStore) DeleteByCategory(category string) error {
	return s.db.Where("category = ?", category).Delete(&model.SystemSetting{}).Error
}

func (s *settingsStore) DeleteSetting(setting *model.SystemSetting) error {
	return s.db.Delete(setting).Error
}

// Query operations

func (s *settingsStore) Count() (int64, error) {
	var count int64
	err := s.db.Model(&model.SystemSetting{}).Count(&count).Error
	return count, err
}

// Batch operations

func (s *settingsStore) BatchUpsert(settings []model.SystemSetting) error {
	if len(settings) == 0 {
		return nil
	}

	// Use transaction for batch upsert
	return s.db.Transaction(func(tx *gorm.DB) error {
		for _, setting := range settings {
			// Try to find existing record
			var existing model.SystemSetting
			err := tx.Where("category = ? AND key = ?", setting.Category, setting.Key).First(&existing).Error
			if err == gorm.ErrRecordNotFound {
				// Create new record
				if err := tx.Create(&setting).Error; err != nil {
					return err
				}
			} else if err != nil {
				return err
			} else {
				// Update existing record
				existing.Value = setting.Value
				existing.ValueType = setting.ValueType
				if err := tx.Save(&existing).Error; err != nil {
					return err
				}
			}
		}
		return nil
	})
}

// Transaction support

func (s *settingsStore) WithTx(tx *gorm.DB) SettingsStore {
	return &settingsStore{db: tx}
}
