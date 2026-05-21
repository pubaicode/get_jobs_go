package model

import "time"

type Config struct {
	ID          uint   `gorm:"primaryKey"`
	ConfigKey   string `gorm:"column:config_key;size:255;uniqueIndex"`
	ConfigValue string `gorm:"column:config_value;type:text"`
	ConfigType  string `gorm:"column:config_type;size:50"`
	Category    string `gorm:"size:50"`
	Description string `gorm:"size:500"`
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

func (Config) TableName() string { return "config" }

type Cookie struct {
	ID          uint   `gorm:"primaryKey"`
	Platform    string `gorm:"size:50"`
	CookieValue string `gorm:"column:cookie_value;type:text"`
	Remark      string `gorm:"type:text"`
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

func (Cookie) TableName() string { return "cookie" }

type AiConfig struct {
	ID        uint      `json:"id" gorm:"primaryKey"`
	Introduce string    `json:"introduce" gorm:"type:text"`
	Prompt    string    `json:"prompt" gorm:"type:text"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

func (AiConfig) TableName() string { return "ai" }
