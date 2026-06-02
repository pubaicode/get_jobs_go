package manager

import (
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/launcher"
	"github.com/go-rod/rod/lib/proto"
)

type LoginStatus string

type LoginStatusChange struct {
	Platform string
	LoggedIn bool
}

type BrowserManager struct {
	mu          sync.RWMutex
	browser     *rod.Browser
	pagePool    map[string]*rod.Page
	loginStatus map[string]bool
	listeners   []chan LoginStatusChange

	monitoringPaused map[string]bool
	stopCh           chan struct{}
	monitors         map[string]bool
}

var DefaultManager *BrowserManager

func init() {
	DefaultManager = NewBrowserManager()
}

func NewBrowserManager() *BrowserManager {
	return &BrowserManager{
		pagePool:         make(map[string]*rod.Page),
		loginStatus:      make(map[string]bool),
		monitoringPaused: make(map[string]bool),
		stopCh:           make(chan struct{}),
		monitors:         make(map[string]bool),
	}
}

func (m *BrowserManager) Init() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.browser != nil {
		_, err := m.browser.GetCookies()
		if err == nil {
			return nil
		}
		log.Printf("Browser connection lost, re-launching: %v", err)
		m.browser = nil
		m.pagePool = make(map[string]*rod.Page)
	}

	userDir := "./.chrome-data"

	// If Chrome is already running with this user data dir, kill it first
	launcher.New().UserDataDir(userDir).Kill()

	u, err := launcher.New().
		UserDataDir(userDir).
		Headless(false).
		Set("start-maximized").
		Set("disable-blink-features", "AutomationControlled").
		Launch()
	if err != nil {
		return fmt.Errorf("failed to launch browser: %w", err)
	}

	m.browser = rod.New().ControlURL(u).
		MustConnect()

	log.Println("Browser launched with persistent profile")

	if err := m.newPage("boss", "https://www.zhipin.com"); err != nil {
		return err
	}
	log.Println("Boss page created")

	return nil
}

func (m *BrowserManager) newPage(name, url string) error {
	page, err := m.browser.Page(proto.TargetCreateTarget{URL: url})
	if err != nil {
		return err
	}
	page.MustSetViewport(1920, 1080, 1, false)
	page.SetExtraHeaders([]string{
		"Accept-Language", "zh-CN,zh;q=0.9",
	})
	m.pagePool[name] = page
	return nil
}

func (m *BrowserManager) GetPage(name string) *rod.Page {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.pagePool[name]
}

func (m *BrowserManager) EnsurePage(name, url string) (*rod.Page, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if p, ok := m.pagePool[name]; ok {
		_, err := p.Info()
		if err == nil {
			return p, nil
		}
		log.Printf("Page %s target lost, creating new one: %v", name, err)
		delete(m.pagePool, name)
	}

	page, err := m.browser.Page(proto.TargetCreateTarget{URL: url})
	if err != nil {
		return nil, err
	}
	m.pagePool[name] = page
	return page, nil
}

func (m *BrowserManager) SetLoginStatus(platform string, loggedIn bool) {
	m.mu.Lock()
	prev := m.loginStatus[platform]
	m.loginStatus[platform] = loggedIn
	listeners := make([]chan LoginStatusChange, len(m.listeners))
	copy(listeners, m.listeners)
	m.mu.Unlock()

	if loggedIn != prev {
		change := LoginStatusChange{Platform: platform, LoggedIn: loggedIn}
		for _, ch := range listeners {
			select {
			case ch <- change:
			default:
			}
		}
	}
}

func (m *BrowserManager) GetLoginStatus(platform string) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.loginStatus[platform]
}

func (m *BrowserManager) AddListener() chan LoginStatusChange {
	ch := make(chan LoginStatusChange, 16)
	m.mu.Lock()
	m.listeners = append(m.listeners, ch)
	m.mu.Unlock()
	return ch
}

func (m *BrowserManager) IsInitialized() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.browser != nil
}

func (m *BrowserManager) SetMonitoringPaused(platform string, paused bool) {
	m.mu.Lock()
	m.monitoringPaused[platform] = paused
	m.mu.Unlock()
}

func (m *BrowserManager) IsMonitoringPaused(platform string) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.monitoringPaused[platform]
}

func (m *BrowserManager) SaveCookiesToDB(platform string) {
	m.mu.RLock()
	page, ok := m.pagePool[platform]
	m.mu.RUnlock()
	if !ok || page == nil {
		return
	}

	cookies, err := page.Cookies(nil)
	if err != nil {
		log.Printf("Failed to get cookies for %s: %v", platform, err)
		return
	}

	data, _ := json.Marshal(cookies)
	log.Printf("Saving %d cookies for %s", len(cookies), platform)
	_ = data
}

func (m *BrowserManager) StartLoginMonitor(platform string, page *rod.Page) {
	m.mu.Lock()
	if m.monitors[platform] {
		m.mu.Unlock()
		return
	}
	m.monitors[platform] = true
	m.mu.Unlock()

	go func() {
		ticker := time.NewTicker(2 * time.Second)
		defer ticker.Stop()
		defer func() {
			m.mu.Lock()
			delete(m.monitors, platform)
			m.mu.Unlock()
		}()
		checkCount := 0
		for {
			select {
			case <-ticker.C:
				checkCount++
				var loggedIn bool
				switch platform {
				case "boss":
					loggedIn = CheckBossLoggedIn(page)
				case "liepin":
					loggedIn = CheckLiepinLoggedIn(page)
					if !loggedIn && checkCount > 5 {
						if err := page.Navigate("https://www.liepin.com/"); err == nil {
							if err := page.WaitLoad(); err == nil {
								loggedIn = CheckLiepinLoggedIn(page)
								if !loggedIn {
									info, _ := page.Info()
									if info != nil {
										u := info.URL
										if strings.Contains(u, "liepin.com") && !strings.Contains(u, "/login") && !strings.Contains(u, "/passport") {
											loggedIn = true
										}
									}
								}
							}
						}
					}
				case "zhilian":
					loggedIn = CheckZhilianLoggedIn(page)
				default:
					loggedIn = false
				}
				if loggedIn {
					m.SetLoginStatus(platform, true)
					return
				}
			case <-m.stopCh:
				return
			}
		}
	}()
}

func (m *BrowserManager) ClearCookies() {
	if m.browser == nil {
		return
	}
	if err := m.browser.SetCookies(nil); err != nil {
		log.Printf("Failed to clear cookies: %v", err)
	}
}

func (m *BrowserManager) Close() {
	close(m.stopCh)
	if m.browser != nil {
		m.browser.Close()
	}
}

func CheckBossLoggedIn(page *rod.Page) bool {
	if page == nil {
		return false
	}

	el, err := page.Timeout(3 * time.Second).Element("li.nav-figure span.label-text")
	if err == nil && el != nil {
		vis, _ := el.Visible()
		if vis {
			return true
		}
	}

	el, err = page.Timeout(3 * time.Second).Element("li.nav-figure")
	if err == nil && el != nil {
		vis, _ := el.Visible()
		if vis {
			return true
		}
	}

	el, err = page.Timeout(3 * time.Second).Element("li.nav-sign a, .btns")
	if err == nil && el != nil {
		text, _ := el.Text()
		if strings.Contains(text, "登录") {
			return false
		}
	}

	return false
}

func CheckLiepinLoggedIn(page *rod.Page) bool {
	if page == nil {
		return false
	}

	info, err := page.Info()
	if err != nil {
		return false
	}
	u := info.URL

	if strings.Contains(u, "/login") || strings.Contains(u, "/passport") || strings.Contains(u, "login.liepin") {
		return false
	}

	if strings.Contains(u, "/personal") || strings.Contains(u, "/account") || strings.Contains(u, "/resume") || strings.Contains(u, "/settings") || strings.Contains(u, "/my") || strings.Contains(u, "/mycenter") || strings.Contains(u, "/member") {
		return true
	}

	el, _ := page.Timeout(3 * time.Second).Element(
		"#header-quick-menu-user-info, img.header-quick-menu-user-photo, " +
			"[class*='user-photo'], [class*='user-avatar'], [class*='header-user'], " +
			"[class*='nav-user'], [class*='login-user'], [class*='has-login']",
	)
	if el != nil {
		return true
	}

	el, _ = page.Timeout(3 * time.Second).Element(
		"#header-quick-menu-login, a[href*='login'], [class*='btn-login'], [class*='login-btn']",
	)
	if el != nil {
		vis, _ := el.Visible()
		if vis {
			return false
		}
	}

	return false
}

func CheckZhilianLoggedIn(page *rod.Page) bool {
	if page == nil {
		return false
	}

	info, err := page.Info()
	if err != nil {
		return false
	}
	url := info.URL

	if strings.Contains(url, "i.zhaopin.com") {
		return true
	}

	if strings.Contains(url, "passport.zhaopin.com") || strings.Contains(url, "/login") {
		return false
	}

	el, err := page.Timeout(3 * time.Second).Element(".user-info, .user-name, .username-text, .header-user-name, .user-photo, .header-user, img.user-avatar, .has-login, .c-login__top, .header-nav__c-login")
	if err == nil && el != nil {
		return true
	}

	if strings.Contains(url, "zhaopin.com") || strings.Contains(url, "www.zhaopin.com") {
		return true
	}

	return false
}
