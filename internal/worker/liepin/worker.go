package liepin

import (
	"context"
	"fmt"
	"log"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/getjobs/server/internal/database"
	"github.com/getjobs/server/internal/model"
	"github.com/go-rod/rod"
)

type ProgressCallback func(msg string, current, total int)

type Config struct {
	Keywords   []string
	City       string
	SalaryCode string
	WaitTime   int
	Debugger   bool
}

type Worker struct {
	mu       sync.RWMutex
	page     *rod.Page
	cfg      *Config
	running  bool
	stopCh   chan struct{}
	progress ProgressCallback
}

func New(cfg *Config) *Worker {
	return &Worker{
		cfg:    cfg,
		stopCh: make(chan struct{}),
	}
}

func (w *Worker) SetPage(page *rod.Page) {
	w.page = page
}

func (w *Worker) SetProgress(cb ProgressCallback) {
	w.progress = cb
}

func (w *Worker) logProgress(msg string, current, total int) {
	if w.progress != nil {
		w.progress(msg, current, total)
	}
	log.Printf("[Liepin] %s (%d/%d)", msg, current, total)
}

func (w *Worker) shouldStop() bool {
	select {
	case <-w.stopCh:
		return true
	default:
		return false
	}
}

type LiepinJobCard struct {
	JobId      int64  `json:"jobId"`
	JobTitle   string `json:"jobTitle"`
	JobLink    string `json:"jobLink"`
	SalaryText string `json:"salaryText"`
	Area       string `json:"area"`
	EduReq     string `json:"eduReq"`
	ExpReq     string `json:"expReq"`
	PubTime    string `json:"pubTime"`
	CompId     int64  `json:"compId"`
	CompName   string `json:"compName"`
	CompInd    string `json:"compIndustry"`
	CompScale  string `json:"compScale"`
	HrId       string `json:"hrId"`
	HrName     string `json:"hrName"`
	HrTitle    string `json:"hrTitle"`
	HrImId     string `json:"hrImId"`
}

func (w *Worker) Start(ctx context.Context, page *rod.Page) error {
	w.mu.Lock()
	w.running = true
	w.page = page
	w.mu.Unlock()

	defer func() {
		w.mu.Lock()
		w.running = false
		w.mu.Unlock()
	}()

	for _, keyword := range w.cfg.Keywords {
		keyword = strings.TrimSpace(keyword)
		if keyword == "" {
			continue
		}

		if w.shouldStop() {
			return nil
		}

		if err := w.submit(keyword); err != nil {
			log.Printf("Liepin keyword %s error: %v", keyword, err)
		}
	}

	w.logProgress("猎聘投递完成", 0, 0)
	return nil
}

func (w *Worker) Stop() {
	select {
	case w.stopCh <- struct{}{}:
	default:
	}
}

func (w *Worker) IsRunning() bool {
	w.mu.RLock()
	defer w.mu.RUnlock()
	return w.running
}

func (w *Worker) IsLoggedIn() bool {
	if w.page == nil {
		return false
	}
	el, _ := w.page.Timeout(3 * time.Second).Element("#header-quick-menu-login")
	if el != nil {
		return false
	}
	el, _ = w.page.Timeout(3 * time.Second).Element("#header-quick-menu-user-info")
	return el != nil
}

func (w *Worker) Login() error {
	if err := w.page.Navigate("https://www.liepin.com/login"); err != nil {
		return err
	}
	w.page.MustWaitLoad()

	qrEl, _ := w.page.Timeout(5 * time.Second).Element(".switch-type-mask-img-box, img[src*='qrcode-btn']")
	if qrEl != nil {
		qrEl.MustClick()
	}

	log.Println("请在浏览器中扫码登录猎聘...")
	return nil
}

func (w *Worker) submit(keyword string) error {
	u := "https://www.liepin.com/zhaopin/"
	params := url.Values{}
	params.Set("key", keyword)
	if w.cfg.City != "" {
		params.Set("dqs", w.cfg.City)
	}
	if w.cfg.SalaryCode != "" {
		params.Set("salary", w.cfg.SalaryCode)
	}
	fmt.Println(">>><>>", params.Encode())
	if err := w.page.Navigate(u + "?" + params.Encode()); err != nil {
		return err
	}
	w.page.MustWaitLoad()
	time.Sleep(3 * time.Second)

	for pageNum := 1; pageNum <= 20; pageNum++ {
		if w.shouldStop() {
			return nil
		}

		w.logProgress(fmt.Sprintf("处理第 %d 页", pageNum), pageNum, 20)

		cards, err := w.page.Elements(".job-card, .job-list-item, [class*='job-card']")
		if err != nil {
			log.Printf("获取猎聘卡片失败: %v", err)
			continue
		}
		for i, card := range cards {
			if w.shouldStop() {
				return nil
			}

			job := w.parseJobCard(card, i+1, len(cards))
			if job == nil {
				continue
			}

			if w.cfg.Debugger {
				w.logProgress(fmt.Sprintf("调试模式, 跳过投递: %s - %s", job.CompName, job.JobTitle), i+1, len(cards))
				continue
			}

			applyBtn, _ := card.Element("a[data-key='im'], button.btn-im, .btn-chat, span:contains('聊一聊')")
			if applyBtn != nil {
				applyBtn.MustClick()
				time.Sleep(2 * time.Second)

				chatHeader, _ := w.page.Timeout(3 * time.Second).Element(".chat-header, [class*='chat'] [class*='close'], i.icon-close")
				if chatHeader != nil {
					chatHeader.MustClick()
				}

				log.Printf("已投递: %s - %s", job.CompName, job.JobTitle)
				database.DB.Model(&model.LiepinJobData{}).
					Where("job_id = ?", job.JobId).
					Update("delivered", 1)
			}
		}

		nextBtn, _ := w.page.Timeout(3 * time.Second).Element(".page-link-next, .pagination-next")
		if nextBtn == nil {
			break
		}
		nextBtn.MustClick()
		time.Sleep(3 * time.Second)
	}

	return nil
}

func (w *Worker) parseJobCard(card *rod.Element, idx, total int) *LiepinJobCard {
	titleEl, _ := card.Element(".job-title, a[href*='job']")
	salaryEl, _ := card.Element(".job-salary, .salary")
	companyEl, _ := card.Element(".company-name, .company")
	areaEl, _ := card.Element(".job-area, .area")
	expEl, _ := card.Element(".job-experience, .experience")
	eduEl, _ := card.Element(".job-education, .education")

	job := &LiepinJobCard{}

	if titleEl != nil {
		job.JobTitle, _ = titleEl.Text()
		href, _ := titleEl.Attribute("href")
		if href != nil {
			if strings.HasPrefix(*href, "/") {
				job.JobLink = "https://www.liepin.com" + *href
			} else {
				job.JobLink = *href
			}
		}
	}
	if salaryEl != nil {
		job.SalaryText, _ = salaryEl.Text()
	}
	if companyEl != nil {
		job.CompName, _ = companyEl.Text()
	}
	if areaEl != nil {
		job.Area, _ = areaEl.Text()
	}
	if expEl != nil {
		job.ExpReq, _ = expEl.Text()
	}
	if eduEl != nil {
		job.EduReq, _ = eduEl.Text()
	}

	if job.JobTitle == "" {
		return nil
	}

	now := time.Now()
	data := model.LiepinJobData{
		JobTitle:      job.JobTitle,
		JobLink:       job.JobLink,
		JobSalaryText: job.SalaryText,
		JobArea:       job.Area,
		JobEduReq:     job.EduReq,
		JobExpReq:     job.ExpReq,
		CompName:      job.CompName,
		Delivered:     0,
		CreateTime:    now,
		UpdateTime:    now,
	}

	var count int64
	database.DB.Model(&model.LiepinJobData{}).
		Where("job_title = ? AND comp_name = ?", job.JobTitle, job.CompName).
		Count(&count)
	if count == 0 {
		database.DB.Create(&data)
	}

	w.logProgress(fmt.Sprintf("[%d/%d] %s - %s", idx, total, job.CompName, job.JobTitle), idx, total)
	return job
}

func randomPause() {
	time.Sleep(time.Duration(800+int(time.Now().UnixNano()%700)) * time.Millisecond)
}
