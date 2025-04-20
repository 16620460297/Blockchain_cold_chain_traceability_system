package api

import (
	"time"
)

// 通用响应结构
type Response struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

// 用户相关
type UserRegisterRequest struct {
	Username    string `json:"username" binding:"required"`
	Password    string `json:"password" binding:"required"`
	RealName    string `json:"real_name" binding:"required"`
	Address     string `json:"address" binding:"required"`
	Contact     string `json:"contact" binding:"required"`
	UserType    int    `json:"user_type" binding:"required"` // 1: 厂家, 2: 经销商/店家, 3: 监管方/消费者
	CompanyName string `json:"company_name,omitempty"`
	LicenseNo   string `json:"license_no,omitempty"`
}

type UserLoginRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

type UserLoginResponse struct {
	Token       string `json:"token"`
	UserID      uint   `json:"user_id"`
	Username    string `json:"username"`
	UserType    int    `json:"user_type"`
	RealName    string `json:"real_name"`
	AuditStatus int    `json:"audit_status"` // 0: 未审核, 1: 已审核
}

// 冷冻品相关
type Product struct {
	ID               uint      `json:"id"`
	SKU              string    `json:"sku"`               // 唯一的sku码
	Name             string    `json:"name"`              // 产品名称
	Brand            string    `json:"brand"`             // 品牌
	Specification    string    `json:"specification"`     // 规格/包装
	ProductionDate   time.Time `json:"production_date"`   // 生产日期
	ExpirationDate   time.Time `json:"expiration_date"`   // 保质期
	BatchNumber      string    `json:"batch_number"`      // 批次号
	ManufacturerID   uint      `json:"manufacturer_id"`   // 生产商ID
	MaterialSource   string    `json:"material_source"`   // 原材料产地
	ProcessLocation  string    `json:"process_location"`  // 加工地点
	ProcessMethod    string    `json:"process_method"`    // 加工方式
	TransportTemp    float64   `json:"transport_temp"`    // 运输温度
	StorageCondition string    `json:"storage_condition"` // 仓储条件
	SafetyTesting    string    `json:"safety_testing"`    // 安全检测结果
	QualityRating    string    `json:"quality_rating"`    // 质量评估
	ImageURL         string    `json:"image_url"`         // 图片链接
	Status           int       `json:"status"`            // 0: 待审核, 1: 已发布
	CreatedAt        time.Time `json:"created_at"`
}

type AddProductRequest struct {
	Name             string  `json:"name" binding:"required"`
	Brand            string  `json:"brand" binding:"required"`
	Specification    string  `json:"specification" binding:"required"`
	ProductionDate   string  `json:"production_date" binding:"required"` // 格式: "2006-01-02"
	ExpirationDate   string  `json:"expiration_date" binding:"required"` // 格式: "2006-01-02"
	BatchNumber      string  `json:"batch_number" binding:"required"`
	MaterialSource   string  `json:"material_source" binding:"required"`
	ProcessLocation  string  `json:"process_location" binding:"required"`
	ProcessMethod    string  `json:"process_method" binding:"required"`
	TransportTemp    float64 `json:"transport_temp" binding:"required"`
	StorageCondition string  `json:"storage_condition" binding:"required"`
	SafetyTesting    string  `json:"safety_testing" binding:"required"`
	QualityRating    string  `json:"quality_rating" binding:"required"`
	ImageBase64      string  `json:"image_base64" binding:"required"` // Base64编码的图片
}

// 物流信息
type LogisticsInfo struct {
	ID                uint      `json:"id"`
	ProductSKU        string    `json:"product_sku" binding:"required"`
	TrackingNo        string    `json:"tracking_no" binding:"required"`
	WarehouseLocation string    `json:"warehouse_location" binding:"required"`
	Temperature       float64   `json:"temperature" binding:"required"`
	Humidity          float64   `json:"humidity" binding:"required"`
	ImageURL          string    `json:"image_url"`
	ImageBase64       string    `json:"image_base64,omitempty"` // 仅用于请求
	OperatorID        uint      `json:"operator_id"`
	OperatorType      int       `json:"operator_type"` // 1: 厂家, 2: 经销商
	CreatedAt         time.Time `json:"created_at"`
}

// 溯源信息
type TraceInfo struct {
	Product    Product            `json:"product"`
	Logistics  []LogisticsInfo    `json:"logistics"`
	Blockchain []BlockchainRecord `json:"blockchain"`
}

// 区块链记录
type BlockchainRecord struct {
	ID           uint      `json:"id"`
	ProductSKU   string    `json:"product_sku"`
	RecordType   int       `json:"record_type"` // 1: 产品创建, 2: 物流更新, 3: 确认交接
	RecordData   string    `json:"record_data"`
	Hash         string    `json:"hash"`
	PreviousHash string    `json:"previous_hash"`
	Timestamp    time.Time `json:"timestamp"`
}

// 交接确认
type TransferConfirmRequest struct {
	ProductSKU string `json:"product_sku" binding:"required"`
	FromUserID uint   `json:"from_user_id" binding:"required"`
	ToUserID   uint   `json:"to_user_id" binding:"required"`
	Remarks    string `json:"remarks"`
}

// 审核请求
type AuditRequest struct {
	ID     uint   `json:"id" binding:"required"`
	Status int    `json:"status" binding:"required"` // 1: 通过, 2: 拒绝
	Remark string `json:"remark"`
}
