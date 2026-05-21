package handler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/getjobs/server/internal/model"
	"github.com/getjobs/server/internal/service"
	"github.com/getjobs/server/pkg/sse"
)

type BossHandler struct {
	svc *service.BossService
	sse *sse.SSEManager
}

func NewBossHandler(svc *service.BossService, sse *sse.SSEManager) *BossHandler {
	return &BossHandler{svc: svc, sse: sse}
}

func (h *BossHandler) Stream(c *gin.Context) {
	h.sse.GetBroker("boss").ServeHTTP(c)
}

func (h *BossHandler) Execute(c *gin.Context) {
	if err := h.svc.Start(c.Request.Context()); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

func (h *BossHandler) Start(c *gin.Context) {
	if err := h.svc.Start(c.Request.Context()); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "message": "Boss delivery started"})
}

func (h *BossHandler) Login(c *gin.Context) {
	if err := h.svc.Login(c.Request.Context()); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "message": "Login initiated"})
}

func (h *BossHandler) Stop(c *gin.Context) {
	if err := h.svc.Stop(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "message": "Boss delivery stopped"})
}

func (h *BossHandler) Logout(c *gin.Context) {
	if err := h.svc.Logout(c.Request.Context()); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "message": "Logged out"})
}

func (h *BossHandler) Status(c *gin.Context) {
	status := h.svc.Status(c.Request.Context())
	c.JSON(http.StatusOK, status)
}

func (h *BossHandler) Stats(c *gin.Context) {
	stats, err := h.svc.GetStats()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, stats)
}

func (h *BossHandler) List(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	size, _ := strconv.Atoi(c.DefaultQuery("size", "20"))
	jobs, total, err := h.svc.GetJobs(nil, page, size)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"items": jobs, "total": total, "page": page, "size": size})
}

func (h *BossHandler) Reload(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"success": true, "message": "Reloaded"})
}

func (h *BossHandler) GetConfig(c *gin.Context) {
	result, err := h.svc.GetConfig()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, result)
}

func (h *BossHandler) UpdateConfig(c *gin.Context) {
	var cfg model.BossConfig
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

func (h *BossHandler) GetOptions(c *gin.Context) {
	optType := c.Param("type")
	opts, err := h.svc.GetOptions(optType)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, opts)
}

func (h *BossHandler) GetBlacklist(c *gin.Context) {
	list, err := h.svc.GetBlacklist()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, list)
}

func (h *BossHandler) AddBlacklist(c *gin.Context) {
	var item model.BossBlacklist
	if err := c.ShouldBindJSON(&item); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if err := h.svc.AddBlacklist(&item); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true})
}

func (h *BossHandler) DeleteBlacklist(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	if err := h.svc.DeleteBlacklist(uint(id)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true})
}

func (h *BossHandler) GetIndustries(c *gin.Context) {
	industries, err := h.svc.GetIndustries()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, industries)
}
