package service

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"sync"

	"github.com/getjobs/server/internal/manager"
	"github.com/getjobs/server/internal/model"
	"github.com/getjobs/server/internal/repository"
	zhilianWorker "github.com/getjobs/server/internal/worker/zhilian"
	"github.com/getjobs/server/pkg/sse"
)

type ZhilianService struct {
	repo   *repository.ZhilianRepo
	sse    *sse.SSEManager
	worker *zhilianWorker.Worker
	mu     sync.RWMutex
}

func NewZhilianService(repo *repository.ZhilianRepo, sse *sse.SSEManager) *ZhilianService {
	return &ZhilianService{repo: repo, sse: sse}
}

func (s *ZhilianService) GetConfig() (interface{}, error) {
	cfg, err := s.repo.GetConfig()
	if err != nil {
		return nil, err
	}

	optionTypes := []string{"city", "salary"}
	options := make(map[string]interface{})
	for _, t := range optionTypes {
		opts, err := s.repo.GetOptions(t)
		if err != nil || opts == nil {
			opts = []model.ZhilianOption{}
		}
		options[t] = opts
	}

	return map[string]interface{}{
		"config":  cfg,
		"options": options,
	}, nil
}

func (s *ZhilianService) SaveConfig(cfg interface{}) error {
	c, ok := cfg.(*model.ZhilianConfig)
	if !ok {
		return fmt.Errorf("invalid config type: expected *model.ZhilianConfig")
	}
	return s.repo.SaveConfig(c)
}

func (s *ZhilianService) UpdateConfig(cfg *model.ZhilianConfig) error {
	return s.repo.SaveConfig(cfg)
}

func (s *ZhilianService) GetOptions(optType string) (interface{}, error) {
	return s.repo.GetOptions(optType)
}

func (s *ZhilianService) GetStats() (interface{}, error) {
	return s.repo.GetStats()
}

func (s *ZhilianService) GetJobs(filter map[string]interface{}, page, size int) (interface{}, int64, error) {
	return s.repo.GetJobs(filter, page, size)
}

func (s *ZhilianService) Start(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.worker != nil && s.worker.IsRunning() {
		return fmt.Errorf("Zhilian worker is already running")
	}

	modelCfg, err := s.repo.GetConfig()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	wCfg := &zhilianWorker.Config{
		Keywords: splitAndTrim(modelCfg.Keywords),
		CityCode: modelCfg.CityCode,
		Salary:   modelCfg.Salary,
		WaitTime: 2,
	}
	w := zhilianWorker.New(wCfg)

	broker := s.sse.GetBroker("zhilian")
	w.SetProgress(func(msg string, current, total int) {
		data, _ := json.Marshal(map[string]interface{}{
			"message": msg,
			"current": current,
			"total":   total,
		})
		broker.Publish("zhilian", string(data))
	})

	bm := manager.DefaultManager
	if !bm.IsInitialized() {
		if err := bm.Init(); err != nil {
			return fmt.Errorf("failed to init browser: %w", err)
		}
	}

	page, err := bm.EnsurePage("zhilian", "https://www.zhaopin.com")
	if err != nil {
		return fmt.Errorf("failed to get zhilian page: %w", err)
	}

	s.worker = w
	go func() {
		if err := w.Start(ctx, page); err != nil {
			log.Printf("Zhilian worker error: %v", err)
		}
	}()

	return nil
}

func (s *ZhilianService) Stop() error {
	s.mu.RLock()
	w := s.worker
	s.mu.RUnlock()
	if w != nil {
		w.Stop()
	}
	return nil
}

func (s *ZhilianService) Login(ctx context.Context) error {
	bm := manager.DefaultManager
	if err := bm.Init(); err != nil {
		return fmt.Errorf("failed to init browser: %w", err)
	}
	page, err := bm.EnsurePage("zhilian", "https://www.zhaopin.com")
	if err != nil {
		return err
	}

	w := zhilianWorker.New(&zhilianWorker.Config{})
	w.SetPage(page)
	if err := w.Login(); err != nil {
		return err
	}
	bm.StartLoginMonitor("zhilian", page)
	return nil
}

func (s *ZhilianService) Logout(ctx context.Context) error {
	manager.DefaultManager.ClearCookies()
	return nil
}

func (s *ZhilianService) LoginStatus(ctx context.Context) map[string]interface{} {
	page := manager.DefaultManager.GetPage("zhilian")
	loggedIn := false
	if page != nil {
		loggedIn = manager.CheckZhilianLoggedIn(page)
	}
	return map[string]interface{}{"loggedIn": loggedIn}
}

func (s *ZhilianService) Status(ctx context.Context) map[string]interface{} {
	s.mu.RLock()
	w := s.worker
	s.mu.RUnlock()

	isRunning := false
	if w != nil {
		isRunning = w.IsRunning()
	}

	page := manager.DefaultManager.GetPage("zhilian")
	isLoggedIn := false
	if page != nil {
		isLoggedIn = manager.CheckZhilianLoggedIn(page)
	}

	return map[string]interface{}{
		"isRunning":  isRunning,
		"isLoggedIn": isLoggedIn,
	}
}

func (s *ZhilianService) Health() map[string]string {
	return map[string]string{"status": "UP"}
}
