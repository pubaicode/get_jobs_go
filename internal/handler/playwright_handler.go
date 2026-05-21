package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/getjobs/server/internal/manager"
)

type PlaywrightHandler struct{}

func NewPlaywrightHandler() *PlaywrightHandler {
	return &PlaywrightHandler{}
}

func (h *PlaywrightHandler) Status(c *gin.Context) {
	bm := manager.DefaultManager
	status := "idle"
	if bm.IsInitialized() {
		status = "running"
	}
	c.JSON(http.StatusOK, gin.H{"status": status})
}

func (h *PlaywrightHandler) TestNavigate(c *gin.Context) {
	bm := manager.DefaultManager
	if !bm.IsInitialized() {
		if err := bm.Init(); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": err.Error()})
			return
		}
	}
	c.JSON(http.StatusOK, gin.H{"success": true})
}
