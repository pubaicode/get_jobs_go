package model

import "time"

type ZhilianConfig struct {
	ID        uint      `json:"id" gorm:"primaryKey"`
	Keywords  string    `json:"keywords" gorm:"type:text"`
	CityCode  string    `json:"cityCode" gorm:"column:cityCode;size:255"`
	Salary    string    `json:"salary" gorm:"size:255"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

func (ZhilianConfig) TableName() string { return "zhilian_config" }

type ZhilianOption struct {
	ID        uint      `json:"id" gorm:"primaryKey"`
	Type      string    `json:"type" gorm:"column:type;size:255"`
	Name      string    `json:"name" gorm:"size:255"`
	Code      string    `json:"code" gorm:"size:255"`
	SortOrder *int      `json:"sortOrder" gorm:"column:sort_order"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

func (ZhilianOption) TableName() string { return "zhilian_option" }

type ZhilianJobData struct {
	ID             uint   `gorm:"primaryKey"`
	JobId          string `gorm:"column:job_id;size:255"`
	JobTitle       string `gorm:"column:job_title;size:500"`
	JobLink        string `gorm:"column:job_link;type:text"`
	Salary         string `gorm:"size:255"`
	Location       string `gorm:"size:255"`
	Experience     string `gorm:"size:255"`
	Degree         string `gorm:"size:255"`
	CompanyName    string `gorm:"column:company_name;size:500"`
	DeliveryStatus string `gorm:"column:delivery_status;size:50;default:'未投递'"`
	CreateTime     time.Time
	UpdateTime     time.Time
}

func (ZhilianJobData) TableName() string { return "zhilian_data" }
