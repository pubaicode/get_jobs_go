package service

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"sync"

	"github.com/getjobs/server/internal/manager"
	"github.com/getjobs/server/internal/model"
	"github.com/getjobs/server/internal/repository"
	bossWorker "github.com/getjobs/server/internal/worker/boss"
	"github.com/getjobs/server/pkg/sse"
)

type BossService struct {
	repo   *repository.BossRepo
	sse    *sse.SSEManager
	worker *bossWorker.Worker
	mu     sync.RWMutex
}

func NewBossService(repo *repository.BossRepo, sse *sse.SSEManager) *BossService {
	return &BossService{repo: repo, sse: sse}
}

func (s *BossService) GetConfig() (interface{}, error) {
	cfg, err := s.repo.GetConfig()
	if err != nil {
		return nil, err
	}

	optionTypes := []string{"city", "industry", "experience", "degree", "salary", "jobType", "scale", "stage"}
	options := make(map[string]interface{})
	for _, t := range optionTypes {
		opts, _ := s.repo.GetOptions(t)
		if opts == nil {
			opts = []model.BossOption{}
		}
		options[t] = opts
	}

	blacklist, _ := s.repo.GetBlacklist()
	if blacklist == nil {
		blacklist = []model.BossBlacklist{}
	}

	return map[string]interface{}{
		"config":    cfg,
		"options":   options,
		"blacklist": blacklist,
	}, nil
}

func (s *BossService) SaveConfig(cfg interface{}) error {
	c, ok := cfg.(*model.BossConfig)
	if !ok {
		return fmt.Errorf("invalid config type: expected *model.BossConfig")
	}
	return s.repo.SaveConfig(c)
}

func (s *BossService) UpdateConfig(cfg *model.BossConfig) error {
	return s.repo.SaveConfig(cfg)
}

func (s *BossService) AddBlacklist(item *model.BossBlacklist) error {
	return s.repo.AddBlacklist(item)
}

func (s *BossService) DeleteBlacklist(id uint) error {
	return s.repo.DeleteBlacklist(id)
}

func (s *BossService) GetOptions(optType string) (interface{}, error) {
	return s.repo.GetOptions(optType)
}

func (s *BossService) GetBlacklist() (interface{}, error) {
	return s.repo.GetBlacklist()
}

func (s *BossService) GetIndustries() (interface{}, error) {
	return s.repo.GetIndustries()
}

func (s *BossService) GetStats() (interface{}, error) {
	return s.repo.GetStats()
}

func (s *BossService) GetJobs(filter map[string]interface{}, page, size int) (interface{}, int64, error) {
	return s.repo.GetJobs(filter, page, size)
}

func (s *BossService) Start(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.worker != nil && s.worker.IsRunning() {
		return fmt.Errorf("Boss worker is already running")
	}

	modelCfg, err := s.repo.GetConfig()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	wCfg := toBossWorkerConfig(modelCfg)
	w := bossWorker.New(wCfg)

	broker := s.sse.GetBroker("boss")
	w.SetProgress(func(msg string, current, total int) {
		data := map[string]interface{}{
			"message": msg,
			"current": current,
			"total":   total,
		}
		if msg == "Boss投递完成" || msg == "用户取消投递" {
			data["type"] = "success"
		}
		jsonData, _ := json.Marshal(data)
		broker.Publish("progress", string(jsonData))
	})

	bm := manager.DefaultManager
	if !bm.IsInitialized() {
		if err := bm.Init(); err != nil {
			return fmt.Errorf("failed to init browser: %w", err)
		}
	}

	page, err := bm.EnsurePage("boss", "https://www.zhipin.com")
	if err != nil {
		return fmt.Errorf("failed to get boss page: %w", err)
	}

	s.worker = w
	go func() {
		if err := w.Start(page); err != nil {
			log.Printf("Boss worker error: %v", err)
		}
	}()

	return nil
}

func (s *BossService) Stop() error {
	s.mu.RLock()
	w := s.worker
	s.mu.RUnlock()
	if w != nil {
		w.Stop()
	}
	return nil
}

func (s *BossService) Login(ctx context.Context) error {
	bm := manager.DefaultManager
	if !bm.IsInitialized() {
		if err := bm.Init(); err != nil {
			return fmt.Errorf("failed to init browser: %w", err)
		}
	}
	page, err := bm.EnsurePage("boss", "https://www.zhipin.com")
	if err != nil {
		return err
	}

	w := bossWorker.New(&bossWorker.Config{})
	w.SetPage(page)
	if err := w.Login(); err != nil {
		return err
	}

	bm.StartLoginMonitor("boss", page)
	return nil
}

func (s *BossService) Logout(ctx context.Context) error {
	manager.DefaultManager.ClearCookies()
	return nil
}

func (s *BossService) Status(ctx context.Context) map[string]interface{} {
	s.mu.RLock()
	w := s.worker
	s.mu.RUnlock()

	isRunning := false
	isLoggedIn := false
	if w != nil {
		isRunning = w.IsRunning()
	}
	page := manager.DefaultManager.GetPage("boss")
	if page != nil {
		w2 := bossWorker.New(&bossWorker.Config{})
		w2.SetPage(page)
		isLoggedIn = w2.IsLoggedIn()
	}
	return map[string]interface{}{
		"isRunning":  isRunning,
		"isLoggedIn": isLoggedIn,
	}
}

func toBossWorkerConfig(cfg *model.BossConfig) *bossWorker.Config {
	wc := &bossWorker.Config{
		Keywords: splitAndTrim(cfg.Keywords),
		CityCode: splitAndTrim(cfg.CityCode),
		Industry: splitAndTrim(cfg.Industry),
		JobType:  splitAndTrim(cfg.JobType),
		SayHi:    cfg.SayHi,
	}
	if cfg.WaitTime != nil {
		wc.WaitTime = *cfg.WaitTime
	}
	if cfg.Debugger != nil {
		wc.Debugger = *cfg.Debugger == 1
	}
	if cfg.FilterDeadHr != nil {
		wc.FilterDeadHr = *cfg.FilterDeadHr == 1
	}
	if cfg.EnableAi != nil {
		wc.EnableAi = *cfg.EnableAi == 1
	}
	if cfg.SendImgResume != nil {
		wc.SendImgResume = *cfg.SendImgResume == 1
	}
	if cfg.ExpectedSalaryMin != nil {
		wc.ExpectedSalaryMin = *cfg.ExpectedSalaryMin
	}
	if cfg.ExpectedSalaryMax != nil {
		wc.ExpectedSalaryMax = *cfg.ExpectedSalaryMax
	}
	wc.DeadStatus = cfg.DeadStatus
	wc.Experience = splitAndTrim(cfg.Experience)
	wc.Degree = splitAndTrim(cfg.Degree)
	wc.Salary = splitAndTrim(cfg.Salary)
	wc.Scale = splitAndTrim(cfg.Scale)
	wc.Stage = splitAndTrim(cfg.Stage)
	wc.DistrictFilter = splitAndTrim(cfg.DistrictFilter)
	return wc
}

func splitAndTrim(s string) []string {
	if s == "" {
		return nil
	}
	parts := strings.Split(s, ",")
	result := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			result = append(result, p)
		}
	}
	if len(result) == 0 {
		return nil
	}
	return result
}
