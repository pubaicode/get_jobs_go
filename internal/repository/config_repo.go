package repository

import (
	"github.com/getjobs/server/internal/database"
	"github.com/getjobs/server/internal/model"
)

type ConfigRepo struct{}

func NewConfigRepo() *ConfigRepo { return &ConfigRepo{} }

func (r *ConfigRepo) GetAll() ([]model.Config, error) {
	var cfgs []model.Config
	err := database.DB.Find(&cfgs).Error
	return cfgs, err
}

func (r *ConfigRepo) GetByKey(key string) (*model.Config, error) {
	var cfg model.Config
	err := database.DB.Where("config_key = ?", key).First(&cfg).Error
	return &cfg, err
}

func (r *ConfigRepo) Upsert(cfg *model.Config) error {
	var existing model.Config
	err := database.DB.Where("config_key = ?", cfg.ConfigKey).First(&existing).Error
	if err != nil {
		return database.DB.Create(cfg).Error
	}
	existing.ConfigValue = cfg.ConfigValue
	existing.ConfigType = cfg.ConfigType
	existing.Category = cfg.Category
	return database.DB.Save(&existing).Error
}
