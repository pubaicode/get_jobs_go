package database

import (
	"log"
	"os"
	"path/filepath"

	"github.com/getjobs/server/internal/model"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var DB *gorm.DB

func Init(dbPath string) {
	dir := filepath.Dir(dbPath)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		log.Fatalf("Failed to create database directory: %v", err)
	}

	cfg := &gorm.Config{
		Logger: logger.Default.LogMode(logger.Warn),
	}
	var err error
	DB, err = gorm.Open(sqlite.Open(dbPath), cfg)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}

	sqlDB, _ := DB.DB()
	sqlDB.SetMaxOpenConns(1)

	migrate()
	seedOptions()
}

func migrate() {
	migrator := DB.Migrator()

	dropAndRecreate := []interface{}{
		&model.BossOption{},
		&model.ZhilianOption{},
		&model.LiepinOption{},
	}
	for _, m := range dropAndRecreate {
		if migrator.HasTable(m) {
			if err := migrator.DropTable(m); err != nil {
				log.Printf("Warning: failed to drop table %T: %v", m, err)
			}
		}
	}

	err := DB.AutoMigrate(
		&model.BossConfig{},
		&model.BossOption{},
		&model.BossIndustry{},
		&model.BossBlacklist{},
		&model.BossJobData{},
		&model.ZhilianConfig{},
		&model.ZhilianOption{},
		&model.ZhilianJobData{},
		&model.LiepinConfig{},
		&model.LiepinOption{},
		&model.LiepinJobData{},
		&model.Config{},
		&model.Cookie{},
		&model.AiConfig{},
	)
	if err != nil {
		log.Fatalf("Failed to migrate database: %v", err)
	}
}

func seedOptions() {
	var count int64
	DB.Model(&model.BossOption{}).Count(&count)
	if count > 0 {
		return
	}

	seedBossOptions()
	seedZhilianOptions()
	seedLiepinOptions()
}

func seedBossOptions() {
	cities := []model.BossOption{
		{Type: "city", Name: "北京", Code: "101010100", SortOrder: intPtr(1)},
		{Type: "city", Name: "上海", Code: "101020100", SortOrder: intPtr(2)},
		{Type: "city", Name: "广州", Code: "101280101", SortOrder: intPtr(3)},
		{Type: "city", Name: "深圳", Code: "101280601", SortOrder: intPtr(4)},
		{Type: "city", Name: "杭州", Code: "101210101", SortOrder: intPtr(5)},
		{Type: "city", Name: "成都", Code: "101270101", SortOrder: intPtr(6)},
		{Type: "city", Name: "南京", Code: "101190101", SortOrder: intPtr(7)},
		{Type: "city", Name: "武汉", Code: "101200101", SortOrder: intPtr(8)},
		{Type: "city", Name: "苏州", Code: "101190401", SortOrder: intPtr(9)},
		{Type: "city", Name: "重庆", Code: "101040100", SortOrder: intPtr(10)},
		{Type: "city", Name: "天津", Code: "101030100", SortOrder: intPtr(11)},
		{Type: "city", Name: "西安", Code: "101110101", SortOrder: intPtr(12)},
		{Type: "city", Name: "长沙", Code: "101250101", SortOrder: intPtr(13)},
		{Type: "city", Name: "合肥", Code: "101220101", SortOrder: intPtr(14)},
		{Type: "city", Name: "郑州", Code: "101180101", SortOrder: intPtr(15)},
		{Type: "city", Name: "东莞", Code: "101281601", SortOrder: intPtr(16)},
		{Type: "city", Name: "青岛", Code: "101120201", SortOrder: intPtr(17)},
		{Type: "city", Name: "厦门", Code: "101230201", SortOrder: intPtr(18)},
		{Type: "city", Name: "福州", Code: "101230101", SortOrder: intPtr(19)},
		{Type: "city", Name: "济南", Code: "101120101", SortOrder: intPtr(20)},
		{Type: "city", Name: "宁波", Code: "101210401", SortOrder: intPtr(21)},
		{Type: "city", Name: "无锡", Code: "101190201", SortOrder: intPtr(22)},
		{Type: "city", Name: "珠海", Code: "101280701", SortOrder: intPtr(23)},
		{Type: "city", Name: "佛山", Code: "101280301", SortOrder: intPtr(24)},
		{Type: "city", Name: "全国", Code: "0", SortOrder: intPtr(0)},
	}
	for _, opt := range cities {
		DB.Create(&opt)
	}

	industries := []model.BossOption{
		{Type: "industry", Name: "不限", Code: "0", SortOrder: intPtr(0)},
		{Type: "industry", Name: "互联网/IT", Code: "1001", SortOrder: intPtr(1)},
		{Type: "industry", Name: "金融", Code: "1002", SortOrder: intPtr(2)},
		{Type: "industry", Name: "医疗健康", Code: "1003", SortOrder: intPtr(3)},
		{Type: "industry", Name: "教育/培训", Code: "1004", SortOrder: intPtr(4)},
		{Type: "industry", Name: "房地产", Code: "1005", SortOrder: intPtr(5)},
		{Type: "industry", Name: "制造业", Code: "1006", SortOrder: intPtr(6)},
		{Type: "industry", Name: "贸易/物流", Code: "1007", SortOrder: intPtr(7)},
		{Type: "industry", Name: "文化/传媒", Code: "1008", SortOrder: intPtr(8)},
		{Type: "industry", Name: "服务业", Code: "1009", SortOrder: intPtr(9)},
		{Type: "industry", Name: "能源/环保", Code: "1010", SortOrder: intPtr(10)},
	}
	for _, opt := range industries {
		DB.Create(&opt)
	}

	experiences := []model.BossOption{
		{Type: "experience", Name: "不限", Code: "0", SortOrder: intPtr(0)},
		{Type: "experience", Name: "应届生", Code: "101", SortOrder: intPtr(1)},
		{Type: "experience", Name: "1年以下", Code: "102", SortOrder: intPtr(2)},
		{Type: "experience", Name: "1-3年", Code: "103", SortOrder: intPtr(3)},
		{Type: "experience", Name: "3-5年", Code: "104", SortOrder: intPtr(4)},
		{Type: "experience", Name: "5-10年", Code: "105", SortOrder: intPtr(5)},
		{Type: "experience", Name: "10年以上", Code: "106", SortOrder: intPtr(6)},
	}
	for _, opt := range experiences {
		DB.Create(&opt)
	}

	degrees := []model.BossOption{
		{Type: "degree", Name: "不限", Code: "0", SortOrder: intPtr(0)},
		{Type: "degree", Name: "大专", Code: "201", SortOrder: intPtr(1)},
		{Type: "degree", Name: "本科", Code: "202", SortOrder: intPtr(2)},
		{Type: "degree", Name: "硕士", Code: "203", SortOrder: intPtr(3)},
		{Type: "degree", Name: "博士", Code: "204", SortOrder: intPtr(4)},
	}
	for _, opt := range degrees {
		DB.Create(&opt)
	}

	salaries := []model.BossOption{
		{Type: "salary", Name: "不限", Code: "0", SortOrder: intPtr(0)},
		{Type: "salary", Name: "3K以下", Code: "1", SortOrder: intPtr(1)},
		{Type: "salary", Name: "3-5K", Code: "2", SortOrder: intPtr(2)},
		{Type: "salary", Name: "5-10K", Code: "3", SortOrder: intPtr(3)},
		{Type: "salary", Name: "10-20K", Code: "4", SortOrder: intPtr(4)},
		{Type: "salary", Name: "20-50K", Code: "5", SortOrder: intPtr(5)},
		{Type: "salary", Name: "50K以上", Code: "6", SortOrder: intPtr(6)},
	}
	for _, opt := range salaries {
		DB.Create(&opt)
	}

	jobTypes := []model.BossOption{
		{Type: "jobType", Name: "不限", Code: "0", SortOrder: intPtr(0)},
		{Type: "jobType", Name: "全职", Code: "1901", SortOrder: intPtr(1)},
		{Type: "jobType", Name: "兼职", Code: "1902", SortOrder: intPtr(2)},
		{Type: "jobType", Name: "实习", Code: "1903", SortOrder: intPtr(3)},
	}
	for _, opt := range jobTypes {
		DB.Create(&opt)
	}

	scales := []model.BossOption{
		{Type: "scale", Name: "不限", Code: "0", SortOrder: intPtr(0)},
		{Type: "scale", Name: "0-20人", Code: "1", SortOrder: intPtr(1)},
		{Type: "scale", Name: "20-99人", Code: "2", SortOrder: intPtr(2)},
		{Type: "scale", Name: "100-499人", Code: "3", SortOrder: intPtr(3)},
		{Type: "scale", Name: "500-999人", Code: "4", SortOrder: intPtr(4)},
		{Type: "scale", Name: "1000-9999人", Code: "5", SortOrder: intPtr(5)},
		{Type: "scale", Name: "10000人以上", Code: "6", SortOrder: intPtr(6)},
	}
	for _, opt := range scales {
		DB.Create(&opt)
	}

	stages := []model.BossOption{
		{Type: "stage", Name: "不限", Code: "0", SortOrder: intPtr(0)},
		{Type: "stage", Name: "未融资", Code: "1", SortOrder: intPtr(1)},
		{Type: "stage", Name: "天使轮", Code: "2", SortOrder: intPtr(2)},
		{Type: "stage", Name: "A轮", Code: "3", SortOrder: intPtr(3)},
		{Type: "stage", Name: "B轮", Code: "4", SortOrder: intPtr(4)},
		{Type: "stage", Name: "C轮", Code: "5", SortOrder: intPtr(5)},
		{Type: "stage", Name: "D轮及以上", Code: "6", SortOrder: intPtr(6)},
		{Type: "stage", Name: "已上市", Code: "7", SortOrder: intPtr(7)},
		{Type: "stage", Name: "不需要融资", Code: "8", SortOrder: intPtr(8)},
	}
	for _, opt := range stages {
		DB.Create(&opt)
	}
}

func seedZhilianOptions() {
	cities := []model.ZhilianOption{
		{Type: "city", Name: "不限", Code: "0", SortOrder: intPtr(0)},
		{Type: "city", Name: "北京", Code: "530", SortOrder: intPtr(1)},
		{Type: "city", Name: "上海", Code: "538", SortOrder: intPtr(2)},
		{Type: "city", Name: "广州", Code: "763", SortOrder: intPtr(3)},
		{Type: "city", Name: "深圳", Code: "765", SortOrder: intPtr(4)},
		{Type: "city", Name: "天津", Code: "531", SortOrder: intPtr(5)},
		{Type: "city", Name: "武汉", Code: "736", SortOrder: intPtr(6)},
		{Type: "city", Name: "西安", Code: "854", SortOrder: intPtr(7)},
		{Type: "city", Name: "成都", Code: "801", SortOrder: intPtr(8)},
		{Type: "city", Name: "大连", Code: "600", SortOrder: intPtr(9)},
		{Type: "city", Name: "长春", Code: "613", SortOrder: intPtr(10)},
		{Type: "city", Name: "沈阳", Code: "599", SortOrder: intPtr(11)},
		{Type: "city", Name: "南京", Code: "635", SortOrder: intPtr(12)},
		{Type: "city", Name: "济南", Code: "702", SortOrder: intPtr(13)},
		{Type: "city", Name: "青岛", Code: "703", SortOrder: intPtr(14)},
		{Type: "city", Name: "杭州", Code: "653", SortOrder: intPtr(15)},
		{Type: "city", Name: "苏州", Code: "639", SortOrder: intPtr(16)},
		{Type: "city", Name: "无锡", Code: "636", SortOrder: intPtr(17)},
		{Type: "city", Name: "宁波", Code: "654", SortOrder: intPtr(18)},
		{Type: "city", Name: "重庆", Code: "551", SortOrder: intPtr(19)},
		{Type: "city", Name: "郑州", Code: "719", SortOrder: intPtr(20)},
		{Type: "city", Name: "长沙", Code: "749", SortOrder: intPtr(21)},
		{Type: "city", Name: "福州", Code: "681", SortOrder: intPtr(22)},
		{Type: "city", Name: "厦门", Code: "682", SortOrder: intPtr(23)},
		{Type: "city", Name: "哈尔滨", Code: "622", SortOrder: intPtr(24)},
		{Type: "city", Name: "石家庄", Code: "565", SortOrder: intPtr(25)},
		{Type: "city", Name: "合肥", Code: "664", SortOrder: intPtr(26)},
		{Type: "city", Name: "太原", Code: "576", SortOrder: intPtr(27)},
		{Type: "city", Name: "昆明", Code: "831", SortOrder: intPtr(28)},
		{Type: "city", Name: "佛山", Code: "768", SortOrder: intPtr(29)},
		{Type: "city", Name: "南昌", Code: "691", SortOrder: intPtr(30)},
		{Type: "city", Name: "贵阳", Code: "822", SortOrder: intPtr(31)},
		{Type: "city", Name: "洛阳", Code: "721", SortOrder: intPtr(32)},
		{Type: "city", Name: "呼和浩特", Code: "587", SortOrder: intPtr(33)},
		{Type: "city", Name: "兰州", Code: "864", SortOrder: intPtr(34)},
		{Type: "city", Name: "乌鲁木齐", Code: "890", SortOrder: intPtr(35)},
		{Type: "city", Name: "南宁", Code: "785", SortOrder: intPtr(36)},
		{Type: "city", Name: "温州", Code: "655", SortOrder: intPtr(37)},
		{Type: "city", Name: "徐州", Code: "637", SortOrder: intPtr(38)},
		{Type: "city", Name: "常州", Code: "638", SortOrder: intPtr(39)},
		{Type: "city", Name: "东莞", Code: "779", SortOrder: intPtr(40)},
		{Type: "city", Name: "香港", Code: "561", SortOrder: intPtr(41)},
	}
	for _, opt := range cities {
		DB.Create(&opt)
	}
}

func seedLiepinOptions() {
	cities := []model.LiepinOption{
		{Type: "city", Name: "全国", Code: "401", SortOrder: intPtr(0)},
		{Type: "city", Name: "北京", Code: "010", SortOrder: intPtr(1)},
		{Type: "city", Name: "上海", Code: "020", SortOrder: intPtr(2)},
		{Type: "city", Name: "天津", Code: "030", SortOrder: intPtr(3)},
		{Type: "city", Name: "重庆", Code: "040", SortOrder: intPtr(4)},
		{Type: "city", Name: "广州", Code: "050020", SortOrder: intPtr(5)},
		{Type: "city", Name: "深圳", Code: "050090", SortOrder: intPtr(6)},
		{Type: "city", Name: "苏州", Code: "060080", SortOrder: intPtr(7)},
		{Type: "city", Name: "南京", Code: "060020", SortOrder: intPtr(8)},
		{Type: "city", Name: "杭州", Code: "070020", SortOrder: intPtr(9)},
		{Type: "city", Name: "大连", Code: "210040", SortOrder: intPtr(10)},
		{Type: "city", Name: "成都", Code: "280020", SortOrder: intPtr(11)},
		{Type: "city", Name: "武汉", Code: "170020", SortOrder: intPtr(12)},
		{Type: "city", Name: "西安", Code: "270020", SortOrder: intPtr(13)},
	}
	for _, opt := range cities {
		DB.Create(&opt)
	}

	salaries := []model.LiepinOption{
		{Type: "salary", Name: "3K以下", Code: "1", SortOrder: intPtr(1)},
		{Type: "salary", Name: "3-5K", Code: "2", SortOrder: intPtr(2)},
		{Type: "salary", Name: "5-10K", Code: "3", SortOrder: intPtr(3)},
		{Type: "salary", Name: "10-20K", Code: "4", SortOrder: intPtr(4)},
	}
	for _, opt := range salaries {
		DB.Create(&opt)
	}
}

func intPtr(i int) *int { return &i }
