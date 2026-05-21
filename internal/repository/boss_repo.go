package repository

import (
	"github.com/getjobs/server/internal/database"
	"github.com/getjobs/server/internal/model"
)

type BossRepo struct{}

func NewBossRepo() *BossRepo { return &BossRepo{} }

func (r *BossRepo) GetConfig() (*model.BossConfig, error) {
	var cfg model.BossConfig
	err := database.DB.First(&cfg).Error
	return &cfg, err
}

func (r *BossRepo) SaveConfig(cfg *model.BossConfig) error {
	return database.DB.Save(cfg).Error
}

func (r *BossRepo) GetOptions(optType string) ([]model.BossOption, error) {
	var opts []model.BossOption
	err := database.DB.Where("type = ?", optType).Order("sort_order ASC").Find(&opts).Error
	return opts, err
}

func (r *BossRepo) GetBlacklist() ([]model.BossBlacklist, error) {
	var list []model.BossBlacklist
	err := database.DB.Order("created_at DESC").Find(&list).Error
	return list, err
}

func (r *BossRepo) AddBlacklist(item *model.BossBlacklist) error {
	return database.DB.Create(item).Error
}

func (r *BossRepo) DeleteBlacklist(id uint) error {
	return database.DB.Delete(&model.BossBlacklist{}, id).Error
}

func (r *BossRepo) GetIndustries() ([]model.BossIndustry, error) {
	var list []model.BossIndustry
	err := database.DB.Find(&list).Error
	return list, err
}

func (r *BossRepo) GetJobs(filter map[string]interface{}, page, size int) ([]model.BossJobData, int64, error) {
	var jobs []model.BossJobData
	var total int64
	query := database.DB.Model(&model.BossJobData{})
	for k, v := range filter {
		query = query.Where(k, v)
	}
	query.Count(&total)
	err := query.Order("id DESC").Offset((page - 1) * size).Limit(size).Find(&jobs).Error
	return jobs, total, err
}

func (r *BossRepo) SaveJob(job *model.BossJobData) error {
	return database.DB.Create(job).Error
}

func (r *BossRepo) UpdateJob(job *model.BossJobData) error {
	return database.DB.Save(job).Error
}

type NameValue struct {
	Name  string `json:"name"`
	Value int64  `json:"value"`
}

type BucketValue struct {
	Bucket string `json:"bucket"`
	Value  int64  `json:"value"`
}

type BossStatsResponse struct {
	Kpi    map[string]interface{} `json:"kpi"`
	Charts BossCharts             `json:"charts"`
}

type BossCharts struct {
	ByStatus      []NameValue   `json:"byStatus"`
	ByCity        []NameValue   `json:"byCity"`
	ByIndustry    []NameValue   `json:"byIndustry"`
	ByCompany     []NameValue   `json:"byCompany"`
	ByExperience  []NameValue   `json:"byExperience"`
	ByDegree      []NameValue   `json:"byDegree"`
	SalaryBuckets []BucketValue `json:"salaryBuckets"`
	DailyTrend    []NameValue   `json:"dailyTrend"`
	HrActivity    []NameValue   `json:"hrActivity"`
}

func (r *BossRepo) GetStats() (*BossStatsResponse, error) {
	var total, delivered, pending, filtered, failed int64
	database.DB.Model(&model.BossJobData{}).Count(&total)
	database.DB.Model(&model.BossJobData{}).Where("delivery_status = ?", "已投递").Count(&delivered)
	database.DB.Model(&model.BossJobData{}).Where("delivery_status = ?", "未投递").Count(&pending)
	database.DB.Model(&model.BossJobData{}).Where("delivery_status = ?", "已过滤").Count(&filtered)
	database.DB.Model(&model.BossJobData{}).Where("delivery_status = ?", "投递失败").Count(&failed)

	byStatus := []NameValue{
		{"已投递", delivered},
		{"未投递", pending},
		{"已过滤", filtered},
		{"投递失败", failed},
	}

	var byCity []NameValue
	database.DB.Model(&model.BossJobData{}).
		Select("location as name, COUNT(*) as value").
		Group("location").Order("value DESC").Limit(10).Scan(&byCity)
	if byCity == nil {
		byCity = []NameValue{}
	}

	var byIndustry []NameValue
	database.DB.Model(&model.BossJobData{}).
		Select("industry as name, COUNT(*) as value").
		Group("industry").Order("value DESC").Limit(10).Scan(&byIndustry)
	if byIndustry == nil {
		byIndustry = []NameValue{}
	}

	var byCompany []NameValue
	database.DB.Model(&model.BossJobData{}).
		Select("company_name as name, COUNT(*) as value").
		Group("company_name").Order("value DESC").Limit(10).Scan(&byCompany)
	if byCompany == nil {
		byCompany = []NameValue{}
	}

	var byExperience []NameValue
	database.DB.Model(&model.BossJobData{}).
		Select("experience as name, COUNT(*) as value").
		Group("experience").Order("value DESC").Scan(&byExperience)
	if byExperience == nil {
		byExperience = []NameValue{}
	}

	var byDegree []NameValue
	database.DB.Model(&model.BossJobData{}).
		Select("degree as name, COUNT(*) as value").
		Group("degree").Order("value DESC").Scan(&byDegree)
	if byDegree == nil {
		byDegree = []NameValue{}
	}

	var salaryBuckets []BucketValue
	database.DB.Raw(`
		SELECT
			CASE
				WHEN CAST(SUBSTR(salary, 1, INSTR(salary, '-') - 1) AS INTEGER) <= 5 THEN '0-5K'
				WHEN CAST(SUBSTR(salary, 1, INSTR(salary, '-') - 1) AS INTEGER) <= 10 THEN '5-10K'
				WHEN CAST(SUBSTR(salary, 1, INSTR(salary, '-') - 1) AS INTEGER) <= 20 THEN '10-20K'
				WHEN CAST(SUBSTR(salary, 1, INSTR(salary, '-') - 1) AS INTEGER) <= 30 THEN '20-30K'
				WHEN CAST(SUBSTR(salary, 1, INSTR(salary, '-') - 1) AS INTEGER) <= 50 THEN '30-50K'
				ELSE '50K+'
			END as bucket,
			COUNT(*) as value
		FROM boss_data
		WHERE salary IS NOT NULL AND salary != '' AND INSTR(salary, '-') > 0
		GROUP BY bucket
		ORDER BY bucket
	`).Scan(&salaryBuckets)
	if salaryBuckets == nil {
		salaryBuckets = []BucketValue{}
	}

	var dailyTrend []NameValue
	database.DB.Raw(`
		SELECT DATE(created_at) as name, COUNT(*) as value
		FROM boss_data
		WHERE created_at IS NOT NULL
		GROUP BY DATE(created_at)
		ORDER BY name DESC LIMIT 30
	`).Scan(&dailyTrend)
	if dailyTrend == nil {
		dailyTrend = []NameValue{}
	}

	var hrActivity []NameValue
	database.DB.Model(&model.BossJobData{}).
		Select("hr_active_status as name, COUNT(*) as value").
		Where("hr_active_status IS NOT NULL AND hr_active_status != ''").
		Group("hr_active_status").Order("value DESC").Scan(&hrActivity)
	if hrActivity == nil {
		hrActivity = []NameValue{}
	}

	return &BossStatsResponse{
		Kpi: map[string]interface{}{
			"total":    total,
			"delivered": delivered,
			"pending":  pending,
			"filtered": filtered,
			"failed":   failed,
		},
		Charts: BossCharts{
			ByStatus:      byStatus,
			ByCity:        byCity,
			ByIndustry:    byIndustry,
			ByCompany:     byCompany,
			ByExperience:  byExperience,
			ByDegree:      byDegree,
			SalaryBuckets: salaryBuckets,
			DailyTrend:    dailyTrend,
			HrActivity:    hrActivity,
		},
	}, nil
}
