package main

import (
	"embed"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/getjobs/server/internal/config"
	"github.com/getjobs/server/internal/database"
	"github.com/getjobs/server/internal/handler"
	"github.com/getjobs/server/internal/manager"
	"github.com/getjobs/server/internal/middleware"
	"github.com/getjobs/server/internal/repository"
	"github.com/getjobs/server/internal/service"
	"github.com/getjobs/server/pkg/sse"
	"github.com/gin-gonic/gin"
)

//go:embed all:out
var embeddedDist embed.FS

func main() {
	cfg := config.Load()

	database.Init(cfg.DatabasePath)

	sseMgr := sse.NewSSEManager()

	bossRepo := repository.NewBossRepo()
	bossSvc := service.NewBossService(bossRepo, sseMgr)
	bossHandler := handler.NewBossHandler(bossSvc, sseMgr)

	zhilianRepo := repository.NewZhilianRepo()
	zhilianSvc := service.NewZhilianService(zhilianRepo, sseMgr)
	zhilianHandler := handler.NewZhilianHandler(zhilianSvc)

	liepinRepo := repository.NewLiepinRepo()
	liepinSvc := service.NewLiepinService(liepinRepo, sseMgr)
	liepinHandler := handler.NewLiepinHandler(liepinSvc)

	configRepo := repository.NewConfigRepo()
	configSvc := service.NewConfigService(configRepo)
	configHandler := handler.NewConfigHandler(configSvc)

	cookieRepo := repository.NewCookieRepo()
	cookieHandler := handler.NewCookieHandler(cookieRepo)

	aiRepo := repository.NewAiRepo()
	aiSvc := service.NewAiService(aiRepo, configRepo)
	aiHandler := handler.NewAiHandler(aiSvc)

	healthHandler := handler.NewHealthHandler()
	resumeHandler := handler.NewResumeHandler()
	playwrightHandler := handler.NewPlaywrightHandler()

	r := gin.Default()
	r.Use(middleware.CORS())

	r.GET("/api/health", healthHandler.Health)

	// Boss routes
	boss := r.Group("/api/boss")
	{
		boss.GET("/stream", bossHandler.Stream)
		boss.POST("/execute", bossHandler.Execute)
		boss.POST("/start", bossHandler.Start)
		boss.POST("/login", bossHandler.Login)
		boss.POST("/stop", bossHandler.Stop)
		boss.POST("/logout", bossHandler.Logout)
		boss.GET("/status", bossHandler.Status)
		boss.GET("/stats", bossHandler.Stats)
		boss.GET("/list", bossHandler.List)
		boss.GET("/reload", bossHandler.Reload)

		bc := boss.Group("/config")
		{
			bc.GET("", bossHandler.GetConfig)
			bc.PUT("", bossHandler.UpdateConfig)
			bc.GET("/options/:type", bossHandler.GetOptions)
			bc.GET("/blacklist", bossHandler.GetBlacklist)
			bc.POST("/blacklist", bossHandler.AddBlacklist)
			bc.DELETE("/blacklist/:id", bossHandler.DeleteBlacklist)
			bc.GET("/industries", bossHandler.GetIndustries)
		}
	}

	// Zhilian routes
	zhilian := r.Group("/api/zhilian")
	{
		zhilian.GET("/config", zhilianHandler.GetConfig)
		zhilian.PUT("/config", zhilianHandler.UpdateConfig)
		zhilian.GET("/config/options/city", zhilianHandler.GetOptions)
		zhilian.GET("/login-status", zhilianHandler.LoginStatus)
		zhilian.POST("/login", zhilianHandler.Login)
		zhilian.POST("/logout", zhilianHandler.Logout)
		zhilian.GET("/cookie", cookieHandler.Get)
		zhilian.POST("/save-cookie", cookieHandler.Save)
		zhilian.GET("/stats", zhilianHandler.Stats)
		zhilian.GET("/list", zhilianHandler.List)
		zhilian.POST("/start", zhilianHandler.Start)
		zhilian.POST("/stop", zhilianHandler.Stop)
		zhilian.GET("/status", zhilianHandler.Status)
		zhilian.GET("/health", zhilianHandler.Health)
	}

	// Liepin routes
	liepin := r.Group("/api/liepin")
	{
		liepin.GET("/config", liepinHandler.GetConfig)
		liepin.PUT("/config", liepinHandler.UpdateConfig)
		liepin.GET("/config/options/:type", liepinHandler.GetOptions)
		liepin.GET("/login-status", liepinHandler.LoginStatus)
		liepin.POST("/login", liepinHandler.Login)
		liepin.POST("/logout", liepinHandler.Logout)
		liepin.GET("/cookie", cookieHandler.Get)
		liepin.POST("/save-cookie", cookieHandler.Save)
		liepin.GET("/stats", liepinHandler.Stats)
		liepin.GET("/list", liepinHandler.List)
		liepin.POST("/start", liepinHandler.Start)
		liepin.POST("/stop", liepinHandler.Stop)
		liepin.GET("/status", liepinHandler.Status)
		liepin.GET("/health", liepinHandler.Health)
	}

	// Bridge BrowserManager login status changes to SSE broker
	go bridgeLoginStatus(sseMgr)

	// Login status SSE
	r.GET("/api/jobs/login-status/stream", func(c *gin.Context) {
		c.Header("Content-Type", "text/event-stream")
		c.Header("Cache-Control", "no-cache")
		c.Header("Connection", "keep-alive")

		broker := sseMgr.GetBroker("login-status")
		ch := broker.Subscribe()
		defer broker.Unsubscribe(ch)

		bm := manager.DefaultManager
		bossLoggedIn := bm.GetLoginStatus("boss")
		liepinLoggedIn := bm.GetLoginStatus("liepin")
		zhilianLoggedIn := bm.GetLoginStatus("zhilian")
		if !liepinLoggedIn {
			if page := bm.GetPage("liepin"); page != nil {
				liepinLoggedIn = manager.CheckLiepinLoggedIn(page)
			}
		}
		if !zhilianLoggedIn {
			if page := bm.GetPage("zhilian"); page != nil {
				zhilianLoggedIn = manager.CheckZhilianLoggedIn(page)
			}
		}
		connectedData, _ := json.Marshal(map[string]bool{
			"bossLoggedIn":    bossLoggedIn,
			"liepinLoggedIn":  liepinLoggedIn,
			"zhilianLoggedIn": zhilianLoggedIn,
		})

		c.Stream(func(w io.Writer) bool {
			if connectedData != nil {
				fmt.Fprintf(w, "event: connected\ndata: %s\n\n", connectedData)
				connectedData = nil
				return true
			}
			select {
			case event, ok := <-ch:
				if !ok {
					return false
				}
				if event.Event != "" {
					fmt.Fprintf(w, "event: %s\n", event.Event)
				}
				fmt.Fprintf(w, "data: %s\n\n", event.Data)
				return true
			case <-time.After(30 * time.Second):
				fmt.Fprintf(w, ": ping\n\n")
				return true
			case <-c.Request.Context().Done():
				return false
			}
		})
	})

	// Config routes
	rg := r.Group("/api/config")
	{
		rg.GET("", configHandler.GetAll)
		rg.POST("", configHandler.Update)
		rg.GET("/:key", configHandler.GetByKey)
		rg.PUT("/:key", configHandler.Update)
		rg.GET("/health", configHandler.Health)
	}

	// Cookie routes
	r.GET("/api/cookie", cookieHandler.Get)
	r.POST("/api/cookie/save", cookieHandler.Save)

	// AI routes
	ai := r.Group("/api/ai")
	{
		ai.GET("/config", aiHandler.GetConfig)
		ai.POST("/config", aiHandler.SaveConfig)
		ai.POST("/config/generate-from-boss", aiHandler.GenerateFromBoss)
		ai.GET("/chat", aiHandler.Chat)
		ai.GET("/health", aiHandler.Health)
	}

	// Resume routes
	resume := r.Group("/api/resume")
	{
		resume.GET("/boss/current", resumeHandler.GetBossResume)
		resume.POST("/optimize", resumeHandler.Optimize)
	}

	// Playwright routes
	r.GET("/api/playwright/status", playwrightHandler.Status)
	r.GET("/api/playwright/test-navigate", playwrightHandler.TestNavigate)

	distFS, _ := fs.Sub(embeddedDist, "out")
	fileServer := http.FileServer(http.FS(distFS))
	r.Use(func(c *gin.Context) {
		// 只处理非 API 路径，且包含文件扩展名或者是根路径
		path := c.Request.URL.Path
		if strings.HasPrefix(path, "/api") {
			c.Next() // API 请求跳过静态文件处理
			return
		}

		// 尝试提供静态文件
		fileServer.ServeHTTP(c.Writer, c.Request)
		// 如果返回 404，说明不是静态文件，继续执行后续路由（NoRoute）
		if c.Writer.Status() == http.StatusNotFound {
			// c.Writer.resp() // 重置响应，以便 NoRoute 重新写入
			c.Next()
			return
		}
		c.Abort() // 已成功提供静态文件，终止后续处理
	})

	// 3. SPA 兜底：所有未匹配的请求（包括前端路由）返回 index.html
	r.NoRoute(func(c *gin.Context) {
		// 注意：这里不应该再使用 c.File，而应该从 embed 中读取
		indexHtml, _ := embeddedDist.ReadFile("out/index.html")
		c.Data(http.StatusOK, "text/html; charset=utf-8", indexHtml)
	})

	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		addr := ":" + cfg.ServerPort
		log.Printf("GetJobs Go server starting on %s", addr)
		if err := r.Run(addr); err != nil {
			log.Fatalf("Server failed: %v", err)
		}
	}()

	<-quit
	log.Println("Shutting down server...")
}

func bridgeLoginStatus(sseMgr *sse.SSEManager) {
	ch := manager.DefaultManager.AddListener()
	for change := range ch {
		data, err := json.Marshal(map[string]interface{}{
			"platform":   change.Platform,
			"isLoggedIn": change.LoggedIn,
			"timestamp":  time.Now().UnixMilli(),
		})
		if err != nil {
			continue
		}
		sseMgr.Publish("login-status", "login-status", string(data))
	}
}
