package repository

import (
	"github.com/getjobs/server/internal/database"
	"github.com/getjobs/server/internal/model"
)

type AiRepo struct{}

func NewAiRepo() *AiRepo { return &AiRepo{} }

func (r *AiRepo) Get() (*model.AiConfig, error) {
	var cfg model.AiConfig
	err := database.DB.First(&cfg).Error
	if err != nil {
		return r.CreateDefault()
	}
	return &cfg, nil
}

func (r *AiRepo) CreateDefault() (*model.AiConfig, error) {
	cfg := &model.AiConfig{
		Introduce: "请在此填写您的技能介绍",
		Prompt:    "请在此填写AI提示词模板",
	}
	err := database.DB.Create(cfg).Error
	return cfg, err
}

func (r *AiRepo) Save(cfg *model.AiConfig) error {
	var existing model.AiConfig
	err := database.DB.First(&existing).Error
	if err != nil {
		cfg.ID = 0
		return database.DB.Create(cfg).Error
	}
	existing.Introduce = cfg.Introduce
	existing.Prompt = cfg.Prompt
	return database.DB.Save(&existing).Error
}
