package handler

import (
	"net/http"

	"github.com/getjobs/server/internal/model"
	"github.com/getjobs/server/internal/service"
	"github.com/gin-gonic/gin"
)

type ConfigHandler struct {
	svc *service.ConfigService
}

func NewConfigHandler(svc *service.ConfigService) *ConfigHandler {
	return &ConfigHandler{svc: svc}
}

func (h *ConfigHandler) GetAll(c *gin.Context) {
	cfgs, err := h.svc.GetAll()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	configList, ok := cfgs.([]model.Config)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "invalid config data"})
		return
	}

	data := make(map[string]string)
	for _, cfg := range configList {
		data[cfg.ConfigKey] = cfg.ConfigValue
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "data": data})
}

func (h *ConfigHandler) GetByKey(c *gin.Context) {
	key := c.Param("key")
	cfg, err := h.svc.GetByKey(key)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "config not found"})
		return
	}
	c.JSON(http.StatusOK, cfg)
}

func (h *ConfigHandler) Update(c *gin.Context) {
	key := c.Param("key")
	if key != "" {
		var body struct {
			ConfigValue string `json:"config_value"`
		}
		if err := c.ShouldBindJSON(&body); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		if err := h.svc.UpdateByKey(key, body.ConfigValue); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"success": true})
		return
	}

	var configs map[string]string
	if err := c.ShouldBindJSON(&configs); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if err := h.svc.BatchUpdate(configs); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true})
}

func (h *ConfigHandler) Health(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"status": "UP"})
}
