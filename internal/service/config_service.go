package service

import (
	"fmt"

	"github.com/getjobs/server/internal/model"
	"github.com/getjobs/server/internal/repository"
)

type ConfigService struct {
	repo *repository.ConfigRepo
}

func NewConfigService(repo *repository.ConfigRepo) *ConfigService {
	return &ConfigService{repo: repo}
}

func (s *ConfigService) GetAll() (interface{}, error) {
	return s.repo.GetAll()
}

func (s *ConfigService) GetByKey(key string) (interface{}, error) {
	return s.repo.GetByKey(key)
}

func (s *ConfigService) UpdateByKey(key string, value string) error {
	return s.repo.Upsert(&model.Config{
		ConfigKey:   key,
		ConfigValue: value,
	})
}

func (s *ConfigService) BatchUpdate(configs map[string]string) error {
	for key, value := range configs {
		if err := s.repo.Upsert(&model.Config{
			ConfigKey:   key,
			ConfigValue: value,
		}); err != nil {
			return fmt.Errorf("failed to update config %s: %w", key, err)
		}
	}
	return nil
}
