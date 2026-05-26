package boss

import (
	"fmt"
	"log"
	"net/url"
	"os"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/getjobs/server/internal/database"
	"github.com/getjobs/server/internal/model"
	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/proto"
)

type NameValue struct {
	Name  string `json:"name"`
	Value int64  `json:"value"`
}

type BucketValue struct {
	Bucket string `json:"bucket"`
	Value  int64  `json:"value"`
}

type ProgressCallback func(msg string, current, total int)

type Config struct {
	Debugger          bool
	WaitTime          int
	Keywords          []string
	CityCode          []string
	DistrictFilter    []string
	Industry          []string
	JobType           []string
	Experience        []string
	Degree            []string
	Salary            []string
	Scale             []string
	Stage             []string
	SayHi             string
	ExpectedSalaryMin int
	ExpectedSalaryMax int
	EnableAi          bool
	SendImgResume     bool
	FilterDeadHr      bool
	DeadStatus        string
}

type BossDetail struct {
	EncryptId       string `json:"encryptId"`
	EncryptUserId   string `json:"encryptUserId"`
	JobName         string `json:"jobName"`
	SalaryDesc      string `json:"salaryDesc"`
	LocationName    string `json:"locationName"`
	ExperienceName  string `json:"experienceName"`
	DegreeName      string `json:"degreeName"`
	PostDescription string `json:"postDescription"`
	Address         string `json:"address"`
	BrandName       string `json:"brandName"`
	IndustryName    string `json:"industryName"`
	StageName       string `json:"stageName"`
	ScaleName       string `json:"scaleName"`
	HrName          string `json:"hrName"`
	HrTitle         string `json:"hrTitle"`
	ActiveTimeDesc  string `json:"activeTimeDesc"`
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
	log.Printf("[BOSS] %s (%d/%d)", msg, current, total)
}

func (w *Worker) shouldStop() bool {
	select {
	case <-w.stopCh:
		return true
	default:
		return false
	}
}

func (w *Worker) Start(bossPage *rod.Page) error {
	w.mu.Lock()
	w.running = true
	w.page = bossPage
	w.mu.Unlock()

	defer func() {
		w.mu.Lock()
		w.running = false
		w.mu.Unlock()
	}()

	var blackCompanies, blackRecruiters, blackJobs []string
	database.DB.Model(&model.BossBlacklist{}).
		Where("type = ?", "company").Pluck("value", &blackCompanies)
	database.DB.Model(&model.BossBlacklist{}).
		Where("type = ?", "recruiter").Pluck("value", &blackRecruiters)
	database.DB.Model(&model.BossBlacklist{}).
		Where("type = ?", "job").Pluck("value", &blackJobs)

	w.logProgress(fmt.Sprintf("黑名单: 公司(%d) 招聘者(%d) 职位(%d)",
		len(blackCompanies), len(blackRecruiters), len(blackJobs)), 0, 0)

	for _, cityCode := range w.cfg.CityCode {
		if w.shouldStop() {
			w.logProgress("用户取消投递", 0, 0)
			return nil
		}
		w.postJobByCity(cityCode, blackCompanies, blackRecruiters, blackJobs)
	}

	w.logProgress("Boss投递完成", 0, 0)
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
	el, err := w.page.Timeout(3 * time.Second).Element("li.nav-figure span.label-text")
	if err == nil && el != nil {
		vis, _ := el.Visible()
		if vis {
			return true
		}
	}
	el, err = w.page.Timeout(3 * time.Second).Element("li.nav-sign a")
	if err == nil && el != nil {
		return false
	}
	return false
}

func (w *Worker) Login() error {
	if err := w.page.Navigate("https://www.zhipin.com/web/user/?ka=header-login"); err != nil {
		return err
	}
	if err := w.page.WaitLoad(); err != nil {
		log.Printf("Login page load warning: %v", err)
	}
	log.Println("请在浏览器中扫码登录 Boss 直聘...")
	return nil
}

func (w *Worker) postJobByCity(cityCode string, blackCompanies, blackRecruiters, blackJobs []string) {
	searchUrl := buildSearchUrl(w.cfg, cityCode)

	for _, keyword := range w.cfg.Keywords {
		if w.shouldStop() {
			w.logProgress("用户取消投递", 0, 0)
			return
		}

		u := searchUrl + "&query=" + url.QueryEscape(keyword)
		w.logProgress(fmt.Sprintf("搜索: %s", keyword), 0, 0)

		if err := w.page.Navigate(u); err != nil {
			log.Printf("Navigate error: %v", err)
			continue
		}

		if !w.waitForJobList(u, keyword) {
			continue
		}

		w.scrollToLoadAll(keyword, u)

		postCount := w.processJobCards(blackCompanies, blackRecruiters, blackJobs)
		if postCount == 0 {
			w.logProgress(fmt.Sprintf("【%s】扫描完毕", keyword), 0, 0)
			return
		}
		w.logProgress(fmt.Sprintf("【%s】投递完毕，共投递 %d 个", keyword, postCount), 0, 0)
	}
}

func (w *Worker) scrollToLoadAll(keyword, searchUrl string) {
	for i := 0; i < 120; i++ {
		if w.shouldStop() {
			return
		}
		footer, err := w.page.Element("div#footer, #footer")
		if err == nil {
			vis, _ := footer.Visible()
			if vis {
				return
			}
		}
		w.page.MustEval("() => window.scrollBy(0, Math.floor(window.innerHeight * 1.5))")
		time.Sleep(500 * time.Millisecond)
	}
}

func (w *Worker) processJobCards(blackCompanies, blackRecruiters, blackJobs []string) int {
	postCount := 0
	w.page.MustEval("() => window.scrollTo(0, 0)")
	time.Sleep(2 * time.Second)

	cards, err := w.page.Elements("ul.rec-job-list li.job-card-box")
	if err != nil {
		log.Printf("无法找到岗位卡片: %v", err)
		return 0
	}
	count := len(cards)
	w.logProgress(fmt.Sprintf("找到 %d 个岗位", count), 0, count)

	if w.cfg.Debugger {
		log.Print("调试模式, 跳过投递")
		return 0
	}
	for i := 0; i < count; i++ {
		if w.shouldStop() {
			return postCount
		}

		currentCards, _ := w.page.Elements("ul.rec-job-list li.job-card-box")
		if i >= len(currentCards) {
			log.Printf("currentCards index out of range:%v %v continue", i, currentCards)
			continue
		}

		card := currentCards[i]
		card.MustClick()
		time.Sleep(2 * time.Second)

		detail := w.parseDetailFromPage()
		if detail == nil {
			log.Printf("无法解析岗位详情: %v", detail)
			continue
		}

		detail.EncryptId = extractEncryptIdFromUrl(w.page)
		w.persistIfNew(detail, blackCompanies, blackRecruiters, blackJobs)

		if w.isDetailFiltered(detail, blackCompanies, blackRecruiters, blackJobs) {
			log.Printf("过滤: %s", detail.JobName)
			continue
		}

		w.logProgress(fmt.Sprintf("正在投递: %s", detail.JobName), i+1, count)
		if w.submitResume(detail) {
			postCount++
		}

		if i >= 5 {
			w.page.MustEval("() => window.scrollBy(0, 140)")
			time.Sleep(time.Second)
		}
	}
	return postCount
}

func (w *Worker) parseDetailFromPage() *BossDetail {
	detail := &BossDetail{}
	page := w.page.Timeout(3 * time.Second)

	if jobNameEl, err := page.Element(".job-name, .job-title"); err == nil {
		detail.JobName, _ = jobNameEl.Text()
	}
	if salaryEl, err := page.Element(".salary, .job-salary"); err == nil {
		detail.SalaryDesc, _ = salaryEl.Text()
	}
	if expEl, err := page.Element(".job-experience, .experience"); err == nil {
		detail.ExperienceName, _ = expEl.Text()
	}
	if degreeEl, err := page.Element(".job-education, .degree"); err == nil {
		detail.DegreeName, _ = degreeEl.Text()
	}
	if locationEl, err := page.Element(".job-location, .location"); err == nil {
		detail.LocationName, _ = locationEl.Text()
	}
	if descEl, err := page.Element(".job-description, .job-detail"); err == nil {
		detail.PostDescription, _ = descEl.Text()
	}
	if companyEl, err := page.Element(".company-name, .brand-name, .company-title"); err == nil {
		detail.BrandName, _ = companyEl.Text()
	}
	if hrEl, err := page.Element(".boss-name, .hr-name"); err == nil {
		detail.HrName, _ = hrEl.Text()
	}
	if hrTitleEl, err := page.Element(".boss-title, .hr-title"); err == nil {
		detail.HrTitle, _ = hrTitleEl.Text()
	}
	if activeEl, err := page.Element(".boss-active-status, .hr-active"); err == nil {
		detail.ActiveTimeDesc, _ = activeEl.Text()
	}

	return detail
}

func (w *Worker) persistIfNew(detail *BossDetail, blackCompanies, blackRecruiters, blackJobs []string) {
	if detail.JobName == "" {
		return
	}

	var cnt int64
	database.DB.Model(&model.BossJobData{}).
		Where("job_name = ? AND company_name = ?", detail.JobName, detail.BrandName).
		Count(&cnt)
	if cnt > 0 {
		return
	}

	filtered := w.isDetailFiltered(detail, blackCompanies, blackRecruiters, blackJobs)

	status := "未投递"
	if filtered {
		status = "已过滤"
	}
	now := time.Now().Format("2006-01-02 15:04:05")

	database.DB.Create(&model.BossJobData{
		JobName:        detail.JobName,
		Salary:         detail.SalaryDesc,
		Location:       detail.LocationName,
		Experience:     detail.ExperienceName,
		Degree:         detail.DegreeName,
		JobDescription: detail.PostDescription,
		CompanyName:    detail.BrandName,
		HrName:         detail.HrName,
		HrPosition:     detail.HrTitle,
		HrActiveStatus: detail.ActiveTimeDesc,
		DeliveryStatus: status,
		CreatedAt:      now,
		UpdatedAt:      now,
	})
}

func (w *Worker) isDetailFiltered(detail *BossDetail, blackCompanies, blackRecruiters, blackJobs []string) bool {
	for _, b := range blackJobs {
		if strings.Contains(detail.JobName, b) {
			return true
		}
	}
	for _, b := range blackCompanies {
		if strings.Contains(detail.BrandName, b) {
			return true
		}
	}
	for _, b := range blackRecruiters {
		if strings.Contains(detail.HrTitle, b) {
			return true
		}
	}
	if w.cfg.FilterDeadHr && strings.Contains(detail.ActiveTimeDesc, "年") {
		return true
	}
	return false
}

func (w *Worker) submitResume(detail *BossDetail) bool {
	chatBtn, err := w.page.Timeout(5 * time.Second).Element("a.btn-startchat, a.op-btn-chat")
	if err != nil {
		chatBtn, err = w.page.Timeout(3 * time.Second).Element("a.more-job-btn")
		if err != nil {
			log.Printf("未找到沟通入口: %s", detail.JobName)
			return false
		}
		href, _ := chatBtn.Attribute("href")
		if href != nil && strings.HasPrefix(*href, "/job_detail/") {
			detailUrl := "https://www.zhipin.com" + *href
			newPage, err := w.page.Browser().Page(proto.TargetCreateTarget{URL: detailUrl})
			if err != nil {
				return false
			}
			defer newPage.Close()
			newPage.MustWaitLoad()
			time.Sleep(2 * time.Second)
			chatBtn, err = newPage.Timeout(5 * time.Second).Element("a.btn-startchat, a.op-btn-chat")
			if err != nil {
				log.Printf("详情页无沟通按钮: %s", detail.JobName)
				return false
			}
			chatBtn.MustClick()
			time.Sleep(2 * time.Second)

			input, _ := newPage.Timeout(5 * time.Second).Element("div#chat-input.chat-input[contenteditable='true'], textarea.input-area")
			if input == nil {
				log.Printf("聊天输入框未出现: %s", detail.JobName)
				return false
			}
			msg := w.cfg.SayHi
			if msg == "" {
				msg = "您好，我对这个职位很感兴趣，方便聊聊吗？"
			}
			input.MustInput(msg)
			sendBtn, _ := newPage.Element("a.btn-send, button.btn-send")
			if sendBtn != nil {
				sendBtn.MustClick()
			}
			log.Printf("已投递: %s - %s", detail.BrandName, detail.JobName)
			w.updateDeliveryStatus(detail, "已投递")
			return true
		}
		return false
	}

	chatBtn.MustClick()
	time.Sleep(2 * time.Second)

	input, _ := w.page.Timeout(5 * time.Second).Element("div#chat-input.chat-input[contenteditable='true'], textarea.input-area")
	if input == nil {
		log.Printf("聊天输入框未出现: %s", detail.JobName)
		return false
	}

	msg := w.cfg.SayHi
	if msg == "" {
		msg = "您好，我对这个职位很感兴趣，方便聊聊吗？"
	}
	input.MustInput(msg)
	sendBtn, _ := w.page.Element("a.btn-send, button.btn-send")
	if sendBtn != nil {
		sendBtn.MustClick()
	}
	w.updateDeliveryStatus(detail, "已投递")
	log.Printf("已投递: %s - %s | %s", detail.BrandName, detail.JobName, detail.SalaryDesc)
	time.Sleep(time.Second)

	closeBtn, _ := w.page.Element("i.icon-close")
	if closeBtn != nil {
		closeBtn.MustClick()
	}

	return true
}

func (w *Worker) updateDeliveryStatus(detail *BossDetail, status string) {
	database.DB.Model(&model.BossJobData{}).
		Where("job_name = ? AND company_name = ?", detail.JobName, detail.BrandName).
		Update("delivery_status", status)
}

func (w *Worker) waitForJobList(searchUrl, keyword string) bool {
	start := time.Now()
	notified := false

	for time.Since(start) < 5*time.Minute {
		if w.shouldStop() {
			return false
		}

		currentUrl, _ := w.page.Eval("() => window.location.href")
		urlStr := currentUrl.Value.String()

		if strings.Contains(urlStr, "/web/passport/zp/security.html") ||
			strings.Contains(urlStr, "_security_check=") {
			if !notified {
				w.logProgress("Boss触发安全验证，请在浏览器中完成...", 0, 0)
				notified = true
			}
			time.Sleep(2 * time.Second)
			continue
		}

		if strings.TrimRight(urlStr, "/") == "https://www.zhipin.com" {
			w.page.Navigate(searchUrl)
			time.Sleep(2 * time.Second)
			continue
		}

		el, err := w.page.Timeout(time.Second).Element("ul.rec-job-list")
		if err == nil {
			vis, _ := el.Visible()
			if vis {
				if notified {
					w.logProgress("安全验证已通过", 0, 0)
				}
				return true
			}
		}

		time.Sleep(time.Second)
	}
	return false
}

func extractEncryptIdFromUrl(page *rod.Page) string {
	u, err := page.Eval("() => window.location.href")
	if err != nil || u == nil {
		return ""
	}
	urlStr := u.Value.String()
	re := regexp.MustCompile(`/job_detail/([^\.]+)`)
	m := re.FindStringSubmatch(urlStr)
	if len(m) >= 2 {
		return m[1]
	}
	return ""
}

func buildSearchUrl(cfg *Config, cityCode string) string {
	base := "https://www.zhipin.com/web/geek/jobs"
	params := url.Values{}
	if cityCode != "" {
		params.Set("city", cityCode)
	}
	if len(cfg.JobType) > 0 {
		for _, v := range cfg.JobType {
			params.Add("jobType", v)
		}
	}
	if len(cfg.Salary) > 0 {
		for _, v := range cfg.Salary {
			params.Add("salary", v)
		}
	}
	if len(cfg.Experience) > 0 {
		for _, v := range cfg.Experience {
			params.Add("experience", v)
		}
	}
	if len(cfg.Degree) > 0 {
		for _, v := range cfg.Degree {
			params.Add("degree", v)
		}
	}
	return base + "?" + params.Encode()
}

func ParseSalaryRange(salary string) (int, int) {
	re := regexp.MustCompile(`(\d+)[Kk]?[-~–—](\d+)[Kk]`)
	m := re.FindStringSubmatch(salary)
	if len(m) >= 3 {
		min, _ := strconv.Atoi(m[1])
		max, _ := strconv.Atoi(m[2])
		return min, max
	}
	return 0, 0
}

func writeRawJson(body string) {
	f, _ := os.OpenFile("target/job.txt", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if f != nil {
		defer f.Close()
		f.WriteString(body + "\n---\n")
	}
}
