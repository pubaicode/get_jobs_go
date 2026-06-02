package zhilian

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
	Keywords []string
	CityCode string
	Salary   string
	WaitTime int
	Debugger bool
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
	log.Printf("[ZhiLian] %s (%d/%d)", msg, current, total)
}

func (w *Worker) shouldStop() bool {
	select {
	case <-w.stopCh:
		return true
	default:
		return false
	}
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

		if err := w.deliverByKeyword(keyword); err != nil {
			log.Printf("Zhilian keyword %s error: %v", keyword, err)
		}
	}

	w.logProgress("智联招聘投递完成", 0, 0)
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
	el, err := w.page.Timeout(3 * time.Second).Element("a.home-header__c-no-login")
	if err == nil && el != nil {
		return false
	}
	el, err = w.page.Timeout(3 * time.Second).Element(".user-info, .user-name")
	if err == nil && el != nil {
		return true
	}
	return false
}

func (w *Worker) Login() error {
	if err := w.page.Navigate("https://www.zhaopin.com"); err != nil {
		return err
	}
	w.page.MustWaitLoad()

	noLoginEl, err := w.page.Timeout(5 * time.Second).Element("a.home-header__c-no-login")
	if err == nil && noLoginEl != nil {
		qrEl, err := w.page.Timeout(3 * time.Second).Element("div.zppp-panel-normal-bar__img")
		if err == nil && qrEl != nil {
			qrEl.MustClick()
		}
	}

	log.Println("请在浏览器中扫码登录智联招聘...")
	return nil
}

func (w *Worker) deliverByKeyword(keyword string) error {
	searchUrl := "https://www.zhaopin.com/sou/jl" + w.cfg.CityCode + "/kw01JG0RO0DG06203E01JG/"
	params := url.Values{}
	params.Set("key", keyword)
	if w.cfg.CityCode != "" {
		params.Set("cityCode", w.cfg.CityCode)
	}
	if w.cfg.Salary != "" {
		params.Set("salary", w.cfg.Salary)
	}
	fmt.Println(">>>>zhilian:", params.Encode())
	if err := w.page.Navigate(searchUrl + "?" + params.Encode()); err != nil {
		return err
	}
	w.page.MustWaitLoad()
	time.Sleep(3 * time.Second)

	for pageNum := 1; pageNum <= 50; pageNum++ {
		if w.shouldStop() {
			return nil
		}

		w.logProgress(fmt.Sprintf("处理第 %d 页", pageNum), pageNum, 50)

		w.deliverCurrentPage(keyword)

		if pageNum < 50 {
			nextBtn, err := w.page.Timeout(3 * time.Second).Element(".pagination-btn-next")
			if err != nil {
				w.logProgress("没有更多页", pageNum, 50)
				break
			}
			vis, _ := nextBtn.Visible()
			if !vis {
				break
			}
			nextBtn.MustClick()
			time.Sleep(3 * time.Second)
		}
	}

	return nil
}

func (w *Worker) deliverCurrentPage(keyword string) {
	items, err := w.page.Elements(".joblist-box__item")
	if err != nil {
		log.Printf("获取岗位列表失败: %v", err)
		return
	}
	itemCount := len(items)
	for i, item := range items {
		if w.shouldStop() {
			return
		}

		titleEl, _ := item.Element("a.jobinfo__name, a[href*='jobdetail']")
		salaryEl, _ := item.Element(".jobinfo__salary")
		companyEl, _ := item.Element("a.companyinfo__name")
		infoEls, _ := item.Elements(".jobinfo__other-info-item")

		job := model.ZhilianJobData{}
		if titleEl != nil {
			job.JobTitle, _ = titleEl.Text()
			href, _ := titleEl.Attribute("href")
			if href != nil {
				if !strings.HasPrefix(*href, "http") {
					job.JobLink = "https://www.zhaopin.com" + *href
				} else {
					job.JobLink = *href
				}
			}
		}
		if salaryEl != nil {
			job.Salary, _ = salaryEl.Text()
		}
		if companyEl != nil {
			job.CompanyName, _ = companyEl.Text()
		}
		for idx, infoEl := range infoEls {
			text, _ := infoEl.Text()
			switch idx {
			case 0:
				job.Location = text
			case 1:
				job.Experience = text
			case 2:
				job.Degree = text
			}
		}
		if job.JobTitle == "" {
			continue
		}

		job.DeliveryStatus = "未投递"
		inserted := w.saveIfNotExists(&job)

		if w.cfg.Debugger {
			job.DeliveryStatus = "已扫描"
		} else {
			applyBtn, _ := item.Element("button.collect-and-apply__btn")
			if applyBtn != nil {
				applyBtn.MustClick()
				time.Sleep(2 * time.Second)
				w.handleDeliveryDialog()
				job.DeliveryStatus = "已投递"
				if !inserted {
					database.DB.Model(&model.ZhilianJobData{}).
						Where("job_id = ?", job.JobId).
						Update("delivery_status", "已投递")
				}
			}
		}

		if !inserted {
			job.DeliveryStatus = "已投递"
		}
		_ = job

		w.logProgress(fmt.Sprintf("[%d/%d] %s", i+1, itemCount, job.JobTitle), i+1, itemCount)
		time.Sleep(time.Duration(500+randInt(500)) * time.Millisecond)
	}
}

func (w *Worker) saveIfNotExists(job *model.ZhilianJobData) bool {
	var count int64
	database.DB.Model(&model.ZhilianJobData{}).
		Where("job_title = ? AND company_name = ?", job.JobTitle, job.CompanyName).
		Count(&count)
	if count > 0 {
		return false
	}
	now := time.Now()
	job.CreateTime = now
	job.UpdateTime = now
	database.DB.Create(job)
	return true
}

func (w *Worker) handleDeliveryDialog() {
	popup, _ := w.page.Timeout(5 * time.Second).Element(".dialog-content, .apply-dialog, .modal")
	if popup != nil {
		confirmBtn, _ := popup.Element("button.btn-primary, a.btn-primary, .confirm-btn")
		if confirmBtn != nil {
			confirmBtn.MustClick()
			time.Sleep(time.Second)
		}

		closeBtn, _ := popup.Element(".close, .dialog-close, i.icon-close")
		if closeBtn != nil {
			closeBtn.MustClick()
		}
	}

	if w.isLimitReached() {
		log.Println("智联招聘: 检测到每日投递上限")
	}
}

func (w *Worker) isLimitReached() bool {
	el, err := w.page.Timeout(time.Second).Element("div.a-job-apply-workflow")
	if err == nil && el != nil {
		text, _ := el.Text()
		if strings.Contains(text, "达到上限") {
			return true
		}
	}
	return false
}

func randInt(max int) int {
	return int(time.Now().UnixNano() % int64(max))
}
