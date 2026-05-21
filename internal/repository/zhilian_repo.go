package repository

import (
	"github.com/getjobs/server/internal/database"
	"github.com/getjobs/server/internal/model"
)

type ZhilianRepo struct{}

func NewZhilianRepo() *ZhilianRepo { return &ZhilianRepo{} }

func (r *ZhilianRepo) GetConfig() (*model.ZhilianConfig, error) {
	var cfg model.ZhilianConfig
	err := database.DB.First(&cfg).Error
	return &cfg, err
}

func (r *ZhilianRepo) SaveConfig(cfg *model.ZhilianConfig) error {
	return database.DB.Save(cfg).Error
}

func (r *ZhilianRepo) GetOptions(optType string) ([]model.ZhilianOption, error) {
	var opts []model.ZhilianOption
	err := database.DB.Where("type = ?", optType).Order("sort_order ASC").Find(&opts).Error
	return opts, err
}

func (r *ZhilianRepo) GetJobs(filter map[string]interface{}, page, size int) ([]model.ZhilianJobData, int64, error) {
	var jobs []model.ZhilianJobData
	var total int64
	query := database.DB.Model(&model.ZhilianJobData{})
	for k, v := range filter {
		query = query.Where(k, v)
	}
	query.Count(&total)
	err := query.Order("id DESC").Offset((page - 1) * size).Limit(size).Find(&jobs).Error
	return jobs, total, err
}

func (r *ZhilianRepo) SaveJob(job *model.ZhilianJobData) error {
	return database.DB.Create(job).Error
}

type ZhilianCharts struct {
	ByStatus      []NameValue   `json:"byStatus"`
	ByCity        []NameValue   `json:"byCity"`
	ByCompany     []NameValue   `json:"byCompany"`
	ByExperience  []NameValue   `json:"byExperience"`
	ByDegree      []NameValue   `json:"byDegree"`
	SalaryBuckets []BucketValue `json:"salaryBuckets"`
	DailyTrend    []NameValue   `json:"dailyTrend"`
}

type ZhilianStatsResponse struct {
	Kpi    map[string]interface{} `json:"kpi"`
	Charts ZhilianCharts          `json:"charts"`
}

func (r *ZhilianRepo) GetStats() (*ZhilianStatsResponse, error) {
	var total, delivered, pending, filtered, failed int64
	database.DB.Model(&model.ZhilianJobData{}).Count(&total)
	database.DB.Model(&model.ZhilianJobData{}).Where("delivery_status = ?", "已投递").Count(&delivered)
	database.DB.Model(&model.ZhilianJobData{}).Where("delivery_status = ?", "未投递").Count(&pending)
	database.DB.Model(&model.ZhilianJobData{}).Where("delivery_status = ?", "已过滤").Count(&filtered)
	database.DB.Model(&model.ZhilianJobData{}).Where("delivery_status = ?", "投递失败").Count(&failed)

	byStatus := []NameValue{
		{"已投递", delivered},
		{"未投递", pending},
		{"已过滤", filtered},
		{"投递失败", failed},
	}

	var byCity []NameValue
	database.DB.Model(&model.ZhilianJobData{}).
		Select("location as name, COUNT(*) as value").
		Group("location").Order("value DESC").Limit(10).Scan(&byCity)
	if byCity == nil {
		byCity = []NameValue{}
	}

	var byCompany []NameValue
	database.DB.Model(&model.ZhilianJobData{}).
		Select("company_name as name, COUNT(*) as value").
		Group("company_name").Order("value DESC").Limit(10).Scan(&byCompany)
	if byCompany == nil {
		byCompany = []NameValue{}
	}

	var byExperience []NameValue
	database.DB.Model(&model.ZhilianJobData{}).
		Select("experience as name, COUNT(*) as value").
		Group("experience").Order("value DESC").Scan(&byExperience)
	if byExperience == nil {
		byExperience = []NameValue{}
	}

	var byDegree []NameValue
	database.DB.Model(&model.ZhilianJobData{}).
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
				ELSE '20K+'
			END as bucket,
			COUNT(*) as value
		FROM zhilian_data
		WHERE salary IS NOT NULL AND salary != '' AND INSTR(salary, '-') > 0
		GROUP BY bucket
		ORDER BY bucket
	`).Scan(&salaryBuckets)
	if salaryBuckets == nil {
		salaryBuckets = []BucketValue{}
	}

	var dailyTrend []NameValue
	database.DB.Raw(`
		SELECT DATE(create_time) as name, COUNT(*) as value
		FROM zhilian_data
		WHERE create_time IS NOT NULL
		GROUP BY DATE(create_time)
		ORDER BY name DESC LIMIT 30
	`).Scan(&dailyTrend)
	if dailyTrend == nil {
		dailyTrend = []NameValue{}
	}

	return &ZhilianStatsResponse{
		Kpi: map[string]interface{}{
			"total": total, "delivered": delivered, "pending": pending,
			"filtered": filtered, "failed": failed,
		},
		Charts: ZhilianCharts{
			ByStatus: byStatus, ByCity: byCity, ByCompany: byCompany,
			ByExperience: byExperience, ByDegree: byDegree,
			SalaryBuckets: salaryBuckets, DailyTrend: dailyTrend,
		},
	}, nil
}
