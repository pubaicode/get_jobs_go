package service

import (
	"github.com/getjobs/server/internal/repository"
)

type CookieService struct {
	repo *repository.CookieRepo
}

func NewCookieService(repo *repository.CookieRepo) *CookieService {
	return &CookieService{repo: repo}
}

func (s *CookieService) GetByPlatform(platform string) (interface{}, error) {
	return s.repo.GetByPlatform(platform)
}

func (s *CookieService) Save(platform, cookieValue, remark string) error {
	return nil
}

func (s *CookieService) Delete(platform string) error {
	return s.repo.Delete(platform)
}
