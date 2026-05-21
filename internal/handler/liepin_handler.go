package handler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/getjobs/server/internal/model"
	"github.com/getjobs/server/internal/service"
)

type LiepinHandler struct {
	svc *service.LiepinService
}

func NewLiepinHandler(svc *service.LiepinService) *LiepinHandler {
	return &LiepinHandler{svc: svc}
}

func (h *LiepinHandler) GetConfig(c *gin.Context) {
	cfg, err := h.svc.GetConfig()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, cfg)
}

func (h *LiepinHandler) UpdateConfig(c *gin.Context) {
	var cfg model.LiepinConfig
	if err := c.ShouldBindJSON(&cfg); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if err := h.svc.UpdateConfig(&cfg); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true})
}

func (h *LiepinHandler) GetOptions(c *gin.Context) {
	optType := c.Param("type")
	opts, err := h.svc.GetOptions(optType)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, opts)
}

func (h *LiepinHandler) LoginStatus(c *gin.Context) {
	status := h.svc.LoginStatus(c.Request.Context())
	c.JSON(http.StatusOK, status)
}

func (h *LiepinHandler) Login(c *gin.Context) {
	if err := h.svc.Login(c.Request.Context()); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "message": "Login initiated"})
}

func (h *LiepinHandler) Logout(c *gin.Context) {
	if err := h.svc.Logout(c.Request.Context()); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "message": "Logged out"})
}

func (h *LiepinHandler) Start(c *gin.Context) {
	if err := h.svc.Start(c.Request.Context()); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "message": "Liepin delivery started"})
}

func (h *LiepinHandler) Stop(c *gin.Context) {
	if err := h.svc.Stop(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "message": "Liepin delivery stopped"})
}

func (h *LiepinHandler) Status(c *gin.Context) {
	status := h.svc.Status(c.Request.Context())
	c.JSON(http.StatusOK, status)
}

func (h *LiepinHandler) Stats(c *gin.Context) {
	stats, err := h.svc.GetStats()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, stats)
}

func (h *LiepinHandler) List(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	size, _ := strconv.Atoi(c.DefaultQuery("size", "20"))
	jobs, total, err := h.svc.GetJobs(nil, page, size)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"items": jobs, "total": total, "page": page, "size": size})
}

func (h *LiepinHandler) Health(c *gin.Context) {
	status := h.svc.Health()
	c.JSON(http.StatusOK, status)
}
