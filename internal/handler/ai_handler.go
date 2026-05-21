package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/getjobs/server/internal/service"
)

type AiHandler struct {
	svc *service.AiService
}

func NewAiHandler(svc *service.AiService) *AiHandler {
	return &AiHandler{svc: svc}
}

func (h *AiHandler) GetConfig(c *gin.Context) {
	cfg, err := h.svc.GetConfig()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": cfg})
}

func (h *AiHandler) SaveConfig(c *gin.Context) {
	var req struct {
		Introduce string `json:"introduce"`
		Prompt    string `json:"prompt"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "invalid request body"})
		return
	}
	if req.Introduce == "" || req.Prompt == "" {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "introduce and prompt cannot be empty"})
		return
	}
	cfg, err := h.svc.SaveConfig(req.Introduce, req.Prompt)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": cfg, "message": "AI配置保存成功"})
}

func (h *AiHandler) GenerateFromBoss(c *gin.Context) {
	cfg, err := h.svc.GenerateFromBoss()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": cfg, "message": "已根据Boss在线简历生成AI配置"})
}

func (h *AiHandler) Chat(c *gin.Context) {
	content := c.Query("content")
	if content == "" {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "content parameter is required"})
		return
	}
	result, err := h.svc.Chat(content)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": result, "message": "AI request successful"})
}

func (h *AiHandler) Health(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"success": true, "service": "AiHandler", "status": "healthy"})
}
