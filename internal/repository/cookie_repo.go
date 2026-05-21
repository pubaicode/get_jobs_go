package repository

import (
	"github.com/getjobs/server/internal/database"
	"github.com/getjobs/server/internal/model"
)

type CookieRepo struct{}

func NewCookieRepo() *CookieRepo { return &CookieRepo{} }

func (r *CookieRepo) GetByPlatform(platform string) (*model.Cookie, error) {
	var cookie model.Cookie
	err := database.DB.Where("platform = ?", platform).First(&cookie).Error
	return &cookie, err
}

func (r *CookieRepo) Save(cookie *model.Cookie) error {
	var existing model.Cookie
	err := database.DB.Where("platform = ?", cookie.Platform).First(&existing).Error
	if err != nil {
		return database.DB.Create(cookie).Error
	}
	existing.CookieValue = cookie.CookieValue
	existing.Remark = cookie.Remark
	return database.DB.Save(&existing).Error
}

func (r *CookieRepo) Delete(platform string) error {
	return database.DB.Where("platform = ?", platform).Delete(&model.Cookie{}).Error
}
