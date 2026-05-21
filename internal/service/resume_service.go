package service

import (
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/getjobs/server/internal/manager"
)

type ResumeService struct{}

func NewResumeService() *ResumeService {
	return &ResumeService{}
}

func (s *ResumeService) FetchBossResume() (string, error) {
	bm := manager.DefaultManager
	if !bm.IsInitialized() {
		return "", fmt.Errorf("浏览器未初始化，请先启动浏览器")
	}

	page := bm.GetPage("boss")
	if page == nil {
		return "", fmt.Errorf("Boss页面未创建")
	}

	if !manager.CheckBossLoggedIn(page) {
		return "", fmt.Errorf("请先登录Boss直聘")
	}

	previousUrl, err := page.Eval("() => window.location.href")
	if err != nil {
		return "", fmt.Errorf("获取当前URL失败: %w", err)
	}
	prevURL := previousUrl.Value.String()

	bm.SetMonitoringPaused("boss", true)
	defer bm.SetMonitoringPaused("boss", false)

	resumeURL := "https://www.zhipin.com/web/geek/resume"
	if err := page.Navigate(resumeURL); err != nil {
		return "", fmt.Errorf("导航到简历页失败: %w", err)
	}

	page.MustWaitLoad()
	time.Sleep(3 * time.Second)

	defer func() {
		if prevURL != "" {
			if err := page.Navigate(prevURL); err != nil {
				log.Printf("恢复Boss页面URL失败: %v", err)
			}
			page.MustWaitLoad()
		}
	}()

	currentURL, err := page.Eval("() => window.location.href")
	if err == nil {
		urlStr := currentURL.Value.String()
		if strings.Contains(urlStr, "/web/passport") {
			return "", fmt.Errorf("Boss登录状态失效，请先重新登录")
		}
	}

	script := `
		() => {
			const normalize = (value) => {
				let text = (value || '').split('\t').join(' ');
				while (text.includes('  ')) text = text.split('  ').join(' ');
				while (text.includes('\n\n\n')) text = text.split('\n\n\n').join('\n\n');
				return text.trim();
			};
			['script', 'style', 'noscript', 'svg', 'canvas'].forEach(selector => {
				document.querySelectorAll(selector).forEach(node => node.remove());
			});
			const candidates = [
				'.resume-box', '.resume-content', '.resume-item', '.geek-resume',
				'.resume-preview', '.resume-detail', '.user-resume', 'main', '#main', 'body'
			];
			for (const selector of candidates) {
				const nodes = Array.from(document.querySelectorAll(selector));
				const text = normalize(nodes.map(node => node.innerText || node.textContent || '').join('\n'));
				if (text.length > 200 && /优势|经历|经验|教育|项目|技能|求职|工作|简历/.test(text)) {
					return text;
				}
			}
			return normalize(document.body.innerText || '');
		}
	`

	val, err := page.Eval(script)
	if err != nil {
		return "", fmt.Errorf("提取简历内容失败: %w", err)
	}

	text := val.Value.String()
	if text == "" || len(text) < 200 {
		return "", fmt.Errorf("未能读取到Boss在线简历内容，请确认Boss简历页已完善且当前账号可访问")
	}

	text = normalizeResumeText(text)
	return text, nil
}

func normalizeResumeText(text string) string {
	result := make([]byte, 0, len(text))
	prevNewline := false
	prevSpace := false
	for i := 0; i < len(text); i++ {
		c := text[i]
		if c == '\n' {
			if prevNewline {
				continue
			}
			prevNewline = true
			result = append(result, c)
			continue
		}
		prevNewline = false
		if c == '\t' || c == ' ' {
			if prevSpace {
				continue
			}
			prevSpace = true
			result = append(result, ' ')
			continue
		}
		prevSpace = false
		result = append(result, c)
	}

	maxLength := 20000
	if len(result) > maxLength {
		return string(result[:maxLength]) + "\n...(Boss在线简历内容过长，已截断)"
	}
	return string(result)
}


