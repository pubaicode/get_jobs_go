package handler

import (
	"encoding/json"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/getjobs/server/internal/manager"
	"github.com/getjobs/server/internal/model"
	"github.com/getjobs/server/internal/repository"
)

type CookieHandler struct {
	repo *repository.CookieRepo
}

func NewCookieHandler(repo *repository.CookieRepo) *CookieHandler {
	return &CookieHandler{repo: repo}
}

func (h *CookieHandler) Get(c *gin.Context) {
	platform := c.Query("platform")
	cookie, err := h.repo.GetByPlatform(platform)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"platform": platform, "cookie_value": ""})
		return
	}
	c.JSON(http.StatusOK, cookie)
}

func (h *CookieHandler) Save(c *gin.Context) {
	platform := c.Query("platform")
	if platform == "" {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "platform is required"})
		return
	}

	page := manager.DefaultManager.GetPage(platform)
	if page == nil {
		c.JSON(http.StatusOK, gin.H{"success": true, "message": "no page found for " + platform})
		return
	}

	cookies, err := page.Cookies(nil)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": err.Error()})
		return
	}

	if len(cookies) == 0 {
		c.JSON(http.StatusOK, gin.H{"success": true, "message": "no cookies found"})
		return
	}

	data, _ := json.Marshal(cookies)
	if err := h.repo.Save(&model.Cookie{
		Platform:    platform,
		CookieValue: string(data),
	}); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "message": "saved"})
}
