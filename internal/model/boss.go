package model

import "time"

type BossConfig struct {
	ID                uint      `json:"id" gorm:"primaryKey"`
	Debugger          *int      `json:"debugger" gorm:"default:1"`
	WaitTime          *int      `json:"waitTime" gorm:"default:5"`
	Keywords          string    `json:"keywords" gorm:"type:text"`
	CityCode          string    `json:"cityCode"`
	DistrictFilter    string    `json:"districtFilter" gorm:"type:text"`
	Industry          string    `json:"industry"`
	JobType           string    `json:"jobType"`
	Experience        string    `json:"experience"`
	Degree            string    `json:"degree"`
	Salary            string    `json:"salary"`
	Scale             string    `json:"scale"`
	Stage             string    `json:"stage"`
	SayHi             string    `json:"sayHi" gorm:"type:text"`
	ExpectedSalaryMin *int      `json:"expectedSalaryMin" gorm:"default:0"`
	ExpectedSalaryMax *int      `json:"expectedSalaryMax" gorm:"default:0"`
	EnableAi          *int      `json:"enableAi" gorm:"default:0"`
	SendImgResume     *int      `json:"sendImgResume" gorm:"default:0"`
	FilterDeadHr      *int      `json:"filterDeadHr" gorm:"default:0"`
	DeadStatus        string    `json:"deadStatus" gorm:"type:text"`
	CreatedAt         time.Time `json:"createdAt"`
	UpdatedAt         time.Time `json:"updatedAt"`
}

func (BossConfig) TableName() string { return "boss_config" }

type BossOption struct {
	ID        uint      `json:"id" gorm:"primaryKey"`
	Type      string    `json:"type" gorm:"column:type;size:255"`
	Name      string    `json:"name" gorm:"size:255"`
	Code      string    `json:"code" gorm:"size:255"`
	SortOrder *int      `json:"sortOrder" gorm:"column:sort_order"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

func (BossOption) TableName() string { return "boss_option" }

type BossIndustry struct {
	ID        uint      `json:"id" gorm:"primaryKey"`
	Name      string    `json:"name" gorm:"size:255"`
	Code      int       `json:"code"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

func (BossIndustry) TableName() string { return "boss_industry" }

type BossBlacklist struct {
	ID        uint      `json:"id" gorm:"primaryKey"`
	Type      string    `json:"type" gorm:"size:50;column:type"`
	Value     string    `json:"value" gorm:"size:500"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

func (BossBlacklist) TableName() string { return "boss_blacklist" }

type BossJobData struct {
	ID                uint   `gorm:"primaryKey"`
	EncryptId         string `gorm:"column:encrypt_id;type:text"`
	EncryptUserId     string `gorm:"column:encrypt_user_id;type:text"`
	CompanyName       string `gorm:"column:company_name;type:text"`
	JobName           string `gorm:"column:job_name;type:text"`
	Salary            string `gorm:"type:text"`
	Location          string `gorm:"type:text"`
	Experience        string `gorm:"type:text"`
	Degree            string `gorm:"type:text"`
	HrName            string `gorm:"column:hr_name;type:text"`
	HrPosition        string `gorm:"column:hr_position;type:text"`
	HrActiveStatus    string `gorm:"column:hr_active_status;type:text"`
	DeliveryStatus    string `gorm:"column:delivery_status;type:text;default:'未投递'"`
	JobDescription    string `gorm:"column:job_description;type:text"`
	JobUrl            string `gorm:"column:job_url;type:text"`
	RecruitmentStatus string `gorm:"column:recruitment_status;type:text"`
	CompanyAddress    string `gorm:"column:company_address;type:text"`
	Industry          string `gorm:"type:text"`
	Introduce         string `gorm:"type:text"`
	FinancingStage    string `gorm:"column:financing_stage;type:text"`
	CompanyScale      string `gorm:"column:company_scale;type:text"`
	CreatedAt         string `gorm:"column:created_at;type:text"`
	UpdatedAt         string `gorm:"column:updated_at;type:text"`
}

func (BossJobData) TableName() string { return "boss_data" }
