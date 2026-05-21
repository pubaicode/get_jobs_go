package model

import "time"

type LiepinConfig struct {
	ID         uint      `json:"id" gorm:"primaryKey"`
	Keywords   string    `json:"keywords" gorm:"type:text"`
	City       string    `json:"city" gorm:"size:255"`
	SalaryCode string    `json:"salaryCode" gorm:"column:salary_code;size:255"`
	CreatedAt  time.Time `json:"createdAt"`
	UpdatedAt  time.Time `json:"updatedAt"`
}

func (LiepinConfig) TableName() string { return "liepin_config" }

type LiepinOption struct {
	ID        uint      `json:"id" gorm:"primaryKey"`
	Type      string    `json:"type" gorm:"column:type;size:255"`
	Name      string    `json:"name" gorm:"size:255"`
	Code      string    `json:"code" gorm:"size:255"`
	SortOrder *int      `json:"sortOrder" gorm:"column:sort_order"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

func (LiepinOption) TableName() string { return "liepin_option" }

type LiepinJobData struct {
	JobId           int64  `gorm:"column:job_id;primaryKey"`
	JobTitle        string `gorm:"column:job_title;size:500"`
	JobLink         string `gorm:"column:job_link;type:text"`
	JobSalaryText   string `gorm:"column:job_salary_text;size:255"`
	JobArea         string `gorm:"column:job_area;size:255"`
	JobEduReq       string `gorm:"column:job_edu_req;size:255"`
	JobExpReq       string `gorm:"column:job_exp_req;size:255"`
	JobPublishTime  string `gorm:"column:job_publish_time;size:255"`
	CompId          int64  `gorm:"column:comp_id"`
	CompName        string `gorm:"column:comp_name;size:500"`
	CompIndustry    string `gorm:"column:comp_industry;size:255"`
	CompScale       string `gorm:"column:comp_scale;size:255"`
	HrId            string `gorm:"column:hr_id;size:255"`
	HrName          string `gorm:"column:hr_name;size:255"`
	HrTitle         string `gorm:"column:hr_title;size:255"`
	HrImId          string `gorm:"column:hr_im_id;size:255"`
	Delivered       int    `gorm:"default:0"`
	CreateTime      time.Time `gorm:"column:create_time"`
	UpdateTime      time.Time `gorm:"column:update_time"`
}

func (LiepinJobData) TableName() string { return "liepin_data" }
