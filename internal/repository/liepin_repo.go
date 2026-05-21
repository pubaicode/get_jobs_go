package repository

import (
	"github.com/getjobs/server/internal/database"
	"github.com/getjobs/server/internal/model"
)

type LiepinRepo struct{}

func NewLiepinRepo() *LiepinRepo { return &LiepinRepo{} }

func (r *LiepinRepo) GetConfig() (*model.LiepinConfig, error) {
	var cfg model.LiepinConfig
	err := database.DB.First(&cfg).Error
	return &cfg, err
}

func (r *LiepinRepo) SaveConfig(cfg *model.LiepinConfig) error {
	return database.DB.Save(cfg).Error
}

func (r *LiepinRepo) GetOptions(optType string) ([]model.LiepinOption, error) {
	var opts []model.LiepinOption
	err := database.DB.Where("type = ?", optType).Order("sort_order ASC").Find(&opts).Error
	return opts, err
}

func (r *LiepinRepo) GetJobs(filter map[string]interface{}, page, size int) ([]model.LiepinJobData, int64, error) {
	var jobs []model.LiepinJobData
	var total int64
	query := database.DB.Model(&model.LiepinJobData{})
	for k, v := range filter {
		query = query.Where(k, v)
	}
	query.Count(&total)
	err := query.Order("job_id DESC").Offset((page - 1) * size).Limit(size).Find(&jobs).Error
	return jobs, total, err
}

func (r *LiepinRepo) SaveJob(job *model.LiepinJobData) error {
	return database.DB.Create(job).Error
}

type LiepinCharts struct {
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

type LiepinStatsResponse struct {
	Kpi    map[string]interface{} `json:"kpi"`
	Charts LiepinCharts           `json:"charts"`
}

func (r *LiepinRepo) GetStats() (*LiepinStatsResponse, error) {
	var total, delivered, pending int64
	database.DB.Model(&model.LiepinJobData{}).Count(&total)
	database.DB.Model(&model.LiepinJobData{}).Where("delivered = ?", 1).Count(&delivered)
	pending = total - delivered

	byStatus := []NameValue{
		{"已投递", delivered},
		{"未投递", pending},
	}

	var byCity []NameValue
	database.DB.Model(&model.LiepinJobData{}).
		Select("job_area as name, COUNT(*) as value").
		Group("job_area").Order("value DESC").Limit(10).Scan(&byCity)
	if byCity == nil {
		byCity = []NameValue{}
	}

	var byIndustry []NameValue
	database.DB.Model(&model.LiepinJobData{}).
		Select("comp_industry as name, COUNT(*) as value").
		Group("comp_industry").Order("value DESC").Limit(10).Scan(&byIndustry)
	if byIndustry == nil {
		byIndustry = []NameValue{}
	}

	var byCompany []NameValue
	database.DB.Model(&model.LiepinJobData{}).
		Select("comp_name as name, COUNT(*) as value").
		Group("comp_name").Order("value DESC").Limit(10).Scan(&byCompany)
	if byCompany == nil {
		byCompany = []NameValue{}
	}

	var byExperience []NameValue
	database.DB.Model(&model.LiepinJobData{}).
		Select("job_exp_req as name, COUNT(*) as value").
		Group("job_exp_req").Order("value DESC").Scan(&byExperience)
	if byExperience == nil {
		byExperience = []NameValue{}
	}

	var byDegree []NameValue
	database.DB.Model(&model.LiepinJobData{}).
		Select("job_edu_req as name, COUNT(*) as value").
		Group("job_edu_req").Order("value DESC").Scan(&byDegree)
	if byDegree == nil {
		byDegree = []NameValue{}
	}

	var salaryBuckets []BucketValue
	database.DB.Raw(`
		SELECT
			CASE
				WHEN CAST(SUBSTR(job_salary_text, 1, INSTR(job_salary_text, '-') - 1) AS INTEGER) <= 5 THEN '0-5K'
				WHEN CAST(SUBSTR(job_salary_text, 1, INSTR(job_salary_text, '-') - 1) AS INTEGER) <= 10 THEN '5-10K'
				WHEN CAST(SUBSTR(job_salary_text, 1, INSTR(job_salary_text, '-') - 1) AS INTEGER) <= 20 THEN '10-20K'
				WHEN CAST(SUBSTR(job_salary_text, 1, INSTR(job_salary_text, '-') - 1) AS INTEGER) <= 30 THEN '20-30K'
				WHEN CAST(SUBSTR(job_salary_text, 1, INSTR(job_salary_text, '-') - 1) AS INTEGER) <= 50 THEN '30-50K'
				ELSE '50K+'
			END as bucket,
			COUNT(*) as value
		FROM liepin_data
		WHERE job_salary_text IS NOT NULL AND job_salary_text != '' AND INSTR(job_salary_text, '-') > 0
		GROUP BY bucket
		ORDER BY bucket
	`).Scan(&salaryBuckets)
	if salaryBuckets == nil {
		salaryBuckets = []BucketValue{}
	}

	var dailyTrend []NameValue
	database.DB.Raw(`
		SELECT DATE(create_time) as name, COUNT(*) as value
		FROM liepin_data
		WHERE create_time IS NOT NULL
		GROUP BY DATE(create_time)
		ORDER BY name DESC LIMIT 30
	`).Scan(&dailyTrend)
	if dailyTrend == nil {
		dailyTrend = []NameValue{}
	}

	var hrActivity []NameValue
	database.DB.Model(&model.LiepinJobData{}).
		Select("hr_title as name, COUNT(*) as value").
		Where("hr_title IS NOT NULL AND hr_title != ''").
		Group("hr_title").Order("value DESC").Scan(&hrActivity)
	if hrActivity == nil {
		hrActivity = []NameValue{}
	}

	return &LiepinStatsResponse{
		Kpi: map[string]interface{}{
			"total": total, "delivered": delivered, "pending": pending,
		},
		Charts: LiepinCharts{
			ByStatus: byStatus, ByCity: byCity, ByIndustry: byIndustry,
			ByCompany: byCompany, ByExperience: byExperience, ByDegree: byDegree,
			SalaryBuckets: salaryBuckets, DailyTrend: dailyTrend, HrActivity: hrActivity,
		},
	}, nil
}
