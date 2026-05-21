package handler

import (
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/getjobs/server/internal/service"
)

type ResumeHandler struct {
	svc *service.ResumeService
}

func NewResumeHandler() *ResumeHandler {
	return &ResumeHandler{svc: service.NewResumeService()}
}

func (h *ResumeHandler) GetBossResume(c *gin.Context) {
	resumeText, err := h.svc.FetchBossResume()
	if err != nil {
		log.Printf("读取Boss在线简历失败: %v", err)
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": "读取Boss在线简历失败: " + err.Error(),
			"data":    nil,
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Boss在线简历读取成功",
		"data": gin.H{
			"resumeText": resumeText,
			"source":     "boss",
		},
	})
}

func (h *ResumeHandler) Optimize(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"success":          true,
		"optimizedResume":  "",
	})
}
