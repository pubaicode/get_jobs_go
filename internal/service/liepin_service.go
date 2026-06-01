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
	liepinWorker "github.com/getjobs/server/internal/worker/liepin"
	"github.com/getjobs/server/pkg/sse"
)

type LiepinService struct {
	repo   *repository.LiepinRepo
	sse    *sse.SSEManager
	worker *liepinWorker.Worker
	mu     sync.RWMutex
}

func NewLiepinService(repo *repository.LiepinRepo, sse *sse.SSEManager) *LiepinService {
	return &LiepinService{repo: repo, sse: sse}
}

func (s *LiepinService) GetConfig() (interface{}, error) {
	cfg, err := s.repo.GetConfig()
	if err != nil {
		return nil, err
	}

	optionTypes := []string{"city", "salary"}
	options := make(map[string]interface{})
	for _, t := range optionTypes {
		opts, err := s.repo.GetOptions(t)
		if err != nil || opts == nil {
			opts = []model.LiepinOption{}
		}
		options[t] = opts
	}

	return map[string]interface{}{
		"config":  cfg,
		"options": options,
	}, nil
}

func (s *LiepinService) SaveConfig(cfg interface{}) error {
	c, ok := cfg.(*model.LiepinConfig)
	if !ok {
		return fmt.Errorf("invalid config type: expected *model.LiepinConfig")
	}
	return s.repo.SaveConfig(c)
}

func (s *LiepinService) UpdateConfig(cfg *model.LiepinConfig) error {
	return s.repo.SaveConfig(cfg)
}

func (s *LiepinService) GetOptions(optType string) (interface{}, error) {
	return s.repo.GetOptions(optType)
}

func (s *LiepinService) GetStats() (interface{}, error) {
	return s.repo.GetStats()
}

func (s *LiepinService) GetJobs(filter map[string]interface{}, page, size int) (interface{}, int64, error) {
	return s.repo.GetJobs(filter, page, size)
}

func (s *LiepinService) Start(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.worker != nil && s.worker.IsRunning() {
		return fmt.Errorf("Liepin worker is already running")
	}

	modelCfg, err := s.repo.GetConfig()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	wCfg := &liepinWorker.Config{
		Keywords:   splitAndTrim(modelCfg.Keywords),
		City:       modelCfg.City,
		SalaryCode: modelCfg.SalaryCode,
		WaitTime:   2,
		Debugger:   modelCfg.Debugger != nil && *modelCfg.Debugger == 1,
	}
	w := liepinWorker.New(wCfg)

	broker := s.sse.GetBroker("liepin")
	w.SetProgress(func(msg string, current, total int) {
		data, _ := json.Marshal(map[string]interface{}{
			"message": msg,
			"current": current,
			"total":   total,
		})
		broker.Publish("liepin", string(data))
	})

	bm := manager.DefaultManager
	if !bm.IsInitialized() {
		if err := bm.Init(); err != nil {
			return fmt.Errorf("failed to init browser: %w", err)
		}
	}

	page, err := bm.EnsurePage("liepin", "https://www.liepin.com")
	if err != nil {
		return fmt.Errorf("failed to get liepin page: %w", err)
	}

	s.worker = w
	go func() {
		if err := w.Start(ctx, page); err != nil {
			log.Printf("Liepin worker error: %v", err)
		}
	}()

	return nil
}

func (s *LiepinService) Stop() error {
	s.mu.RLock()
	w := s.worker
	s.mu.RUnlock()
	if w != nil {
		w.Stop()
	}
	return nil
}

func (s *LiepinService) Login(ctx context.Context) error {
	bm := manager.DefaultManager
	if !bm.IsInitialized() {
		if err := bm.Init(); err != nil {
			return fmt.Errorf("failed to init browser: %w", err)
		}
	}
	page, err := bm.EnsurePage("liepin", "https://www.liepin.com")
	if err != nil {
		return err
	}

	w := liepinWorker.New(&liepinWorker.Config{})
	w.SetPage(page)
	if err := w.Login(); err != nil {
		return err
	}

	bm.StartLoginMonitor("liepin", page)
	return nil
}

func (s *LiepinService) Logout(ctx context.Context) error {
	manager.DefaultManager.ClearCookies()
	return nil
}

func (s *LiepinService) LoginStatus(ctx context.Context) map[string]interface{} {
	page := manager.DefaultManager.GetPage("liepin")
	loggedIn := false
	if page != nil {
		loggedIn = manager.CheckLiepinLoggedIn(page)
	}
	return map[string]interface{}{"loggedIn": loggedIn}
}

func (s *LiepinService) Status(ctx context.Context) map[string]interface{} {
	s.mu.RLock()
	w := s.worker
	s.mu.RUnlock()

	isRunning := false
	if w != nil {
		isRunning = w.IsRunning()
	}

	page := manager.DefaultManager.GetPage("liepin")
	isLoggedIn := false
	if page != nil {
		isLoggedIn = manager.CheckLiepinLoggedIn(page)
	}

	return map[string]interface{}{
		"isRunning":  isRunning,
		"isLoggedIn": isLoggedIn,
	}
}

func (s *LiepinService) Health() map[string]string {
	return map[string]string{"status": "UP"}
}
