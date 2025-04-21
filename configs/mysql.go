package configs

import (
	"fmt"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"time"
)

type MySQLConfig struct {
	Host     string
	Port     int
	User     string
	Password string
	DBName   string
}

var GlobalMySQLConfig = MySQLConfig{
	Host:     "localhost",
	Port:     3306,
	User:     "root",
	Password: "20040712",
	DBName:   "cold_chain",
}

var DB *gorm.DB

func InitMySQL() error {
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=utf8mb4&parseTime=True&loc=Local",
		GlobalMySQLConfig.User,
		GlobalMySQLConfig.Password,
		GlobalMySQLConfig.Host,
		GlobalMySQLConfig.Port,
		GlobalMySQLConfig.DBName)

	var err error
	DB, err = gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		return err
	}

	sqlDB, err := DB.DB()
	if err != nil {
		return err
	}

	// 设置连接池
	sqlDB.SetMaxIdleConns(10)           // 设置空闲连接池中连接的最大数量
	sqlDB.SetMaxOpenConns(100)          // 设置打开数据库连接的最大数量
	sqlDB.SetConnMaxLifetime(time.Hour) // 设置了连接可复用的最大时间

	// 自动迁移表结构
	err = AutoMigrateTables()
	if err != nil {
		return err
	}

	return nil
}

// 模型定义
type User struct {
	gorm.Model
	Username    string `gorm:"uniqueIndex;size:50;not null"`
	Password    string `gorm:"size:100;not null"`
	RealName    string `gorm:"size:50;not null"`
	Address     string `gorm:"size:200;not null"`
	Contact     string `gorm:"size:50;not null"`
	UserType    int    `gorm:"not null"` // 1: 厂家, 2: 经销商/店家, 3: 监管方/消费者, 4: 管理员
	CompanyName string `gorm:"size:100"`
	LicenseNo   string `gorm:"size:50"`
	AuditStatus int    `gorm:"default:0"` // 0: 未审核, 1: 已审核通过, 2: 已拒绝
	AuditRemark string
}

type ProductInfo struct {
	gorm.Model
	SKU              string    `gorm:"uniqueIndex;size:50;not null"`
	Name             string    `gorm:"size:100;not null"`
	Brand            string    `gorm:"size:50;not null"`
	Specification    string    `gorm:"size:100;not null"`
	ProductionDate   time.Time `gorm:"not null"`
	ExpirationDate   time.Time `gorm:"not null"`
	BatchNumber      string    `gorm:"size:50;not null"`
	ManufacturerID   uint      `gorm:"not null"`
	MaterialSource   string    `gorm:"size:200;not null"`
	ProcessLocation  string    `gorm:"size:200;not null"`
	ProcessMethod    string    `gorm:"size:200;not null"`
	TransportTemp    float64   `gorm:"not null"`
	StorageCondition string    `gorm:"size:200;not null"`
	SafetyTesting    string    `gorm:"size:500;not null"`
	QualityRating    string    `gorm:"size:50;not null"`
	ImageURL         string    `gorm:"size:500;not null"`
	Status           int       `gorm:"default:0"` // 0: 待审核, 1: 已发布
	AuditRemark      string
}

type LogisticsRecord struct {
	gorm.Model
	ProductSKU        string  `gorm:"size:50;not null;index"`
	TrackingNo        string  `gorm:"size:50;not null"`
	WarehouseLocation string  `gorm:"size:200;not null"`
	Temperature       float64 `gorm:"not null"`
	Humidity          float64 `gorm:"not null"`
	ImageURL          string  `gorm:"size:500;not null"`
	OperatorID        uint    `gorm:"not null"`
	OperatorType      int     `gorm:"not null"` // 1: 厂家, 2: 经销商
}

type TransferRecord struct {
	gorm.Model
	ProductSKU string `gorm:"size:50;not null;index"`
	FromUserID uint   `gorm:"not null"`
	ToUserID   uint   `gorm:"not null"`
	Remarks    string
	Status     int `gorm:"default:0"` // 0: 待确认, 1: 已确认
}

type BlockchainLog struct {
	gorm.Model
	ProductSKU   string `gorm:"size:50;not null;index"`
	RecordType   int    `gorm:"not null"` // 1: 产品创建, 2: 物流更新, 3: 确认交接
	RecordData   string `gorm:"type:text;not null"`
	Hash         string `gorm:"size:256;not null"`
	PreviousHash string `gorm:"size:256"`
	BlockHeight  int64
}

func AutoMigrateTables() error {
	return DB.AutoMigrate(
		&User{},
		&ProductInfo{},
		&LogisticsRecord{},
		&TransferRecord{},
		&BlockchainLog{},
	)
}
