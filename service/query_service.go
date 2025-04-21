package service

import (
	"back_Blockchain_cold_chain_traceability_system/api"
	"back_Blockchain_cold_chain_traceability_system/configs"
	"github.com/gin-gonic/gin"
	"net/http"
)

// QueryService 实现查询相关功能
type QueryService struct{}

// TraceProduct 追溯产品信息
func (s *QueryService) TraceProduct(c *gin.Context) {
	sku := c.Query("sku")
	if sku == "" {
		c.JSON(http.StatusBadRequest, api.Response{
			Code:    400,
			Message: "请提供产品SKU",
		})
		return
	}

	// 查询产品基本信息
	var product configs.ProductInfo
	result := configs.DB.Where("sku = ? AND status = 1", sku).First(&product)
	if result.Error != nil {
		c.JSON(http.StatusNotFound, api.Response{
			Code:    404,
			Message: "产品不存在或未上架",
		})
		return
	}

	// 查询物流信息
	var logistics []struct {
		configs.LogisticsRecord
		OperatorName string `json:"operator_name"`
		OperatorType string `json:"operator_type_name"`
	}

	result = configs.DB.Table("logistics_records").
		Select("logistics_records.*, users.real_name as operator_name, CASE logistics_records.operator_type WHEN 1 THEN '厂家' WHEN 2 THEN '经销商' ELSE '未知' END as operator_type_name").
		Joins("JOIN users ON logistics_records.operator_id = users.id").
		Where("logistics_records.product_sku = ?", sku).
		Order("logistics_records.created_at").
		Find(&logistics)

	if result.Error != nil {
		c.JSON(http.StatusInternalServerError, api.Response{
			Code:    500,
			Message: "查询物流信息失败: " + result.Error.Error(),
		})
		return
	}

	// 查询区块链记录
	var blockchain []configs.BlockchainLog
	result = configs.DB.Where("product_sku = ?", sku).
		Order("created_at").
		Find(&blockchain)

	if result.Error != nil {
		c.JSON(http.StatusInternalServerError, api.Response{
			Code:    500,
			Message: "查询区块链记录失败: " + result.Error.Error(),
		})
		return
	}

	// 查询生产商信息
	var manufacturer configs.User
	result = configs.DB.Select("id, real_name, company_name, address, contact").
		Where("id = ?", product.ManufacturerID).
		First(&manufacturer)

	// 查询交接记录
	var transfers []struct {
		configs.TransferRecord
		FromUserName string `json:"from_user_name"`
		ToUserName   string `json:"to_user_name"`
	}

	result = configs.DB.Table("transfer_records").
		Select("transfer_records.*, u1.real_name as from_user_name, u2.real_name as to_user_name").
		Joins("JOIN users u1 ON transfer_records.from_user_id = u1.id").
		Joins("JOIN users u2 ON transfer_records.to_user_id = u2.id").
		Where("transfer_records.product_sku = ? AND transfer_records.status = 1", sku).
		Order("transfer_records.created_at").
		Find(&transfers)

	// 构造溯源信息返回
	traceInfo := gin.H{
		"product": gin.H{
			"sku":               product.SKU,
			"name":              product.Name,
			"brand":             product.Brand,
			"specification":     product.Specification,
			"production_date":   product.ProductionDate.Format("2006-01-02"),
			"expiration_date":   product.ExpirationDate.Format("2006-01-02"),
			"batch_number":      product.BatchNumber,
			"material_source":   product.MaterialSource,
			"process_location":  product.ProcessLocation,
			"process_method":    product.ProcessMethod,
			"transport_temp":    product.TransportTemp,
			"storage_condition": product.StorageCondition,
			"safety_testing":    product.SafetyTesting,
			"quality_rating":    product.QualityRating,
			"image_url":         product.ImageURL,
			"manufacturer":      manufacturer,
		},
		"logistics":  logistics,
		"transfers":  transfers,
		"blockchain": blockchain,
	}

	c.JSON(http.StatusOK, api.Response{
		Code:    200,
		Message: "获取产品溯源信息成功",
		Data:    traceInfo,
	})
}

// VerifyProduct 验证产品真伪
func (s *QueryService) VerifyProduct(c *gin.Context) {
	sku := c.Query("sku")
	if sku == "" {
		c.JSON(http.StatusBadRequest, api.Response{
			Code:    400,
			Message: "请提供产品SKU",
		})
		return
	}

	// 查询区块链记录
	var blockchainCount int64
	result := configs.DB.Model(&configs.BlockchainLog{}).
		Where("product_sku = ?", sku).
		Count(&blockchainCount)

	if result.Error != nil {
		c.JSON(http.StatusInternalServerError, api.Response{
			Code:    500,
			Message: "验证产品失败: " + result.Error.Error(),
		})
		return
	}

	// 查询产品是否存在
	var productCount int64
	result = configs.DB.Model(&configs.ProductInfo{}).
		Where("sku = ? AND status = 1", sku).
		Count(&productCount)

	if result.Error != nil {
		c.JSON(http.StatusInternalServerError, api.Response{
			Code:    500,
			Message: "验证产品失败: " + result.Error.Error(),
		})
		return
	}

	if productCount == 0 {
		c.JSON(http.StatusOK, api.Response{
			Code:    200,
			Message: "验证完成",
			Data: gin.H{
				"authentic": false,
				"message":   "产品不存在或未上架",
			},
		})
		return
	}

	if blockchainCount == 0 {
		c.JSON(http.StatusOK, api.Response{
			Code:    200,
			Message: "验证完成",
			Data: gin.H{
				"authentic": false,
				"message":   "产品没有区块链记录，可能是假冒产品",
			},
		})
		return
	}

	c.JSON(http.StatusOK, api.Response{
		Code:    200,
		Message: "验证完成",
		Data: gin.H{
			"authentic": true,
			"message":   "产品验证通过，是正品",
		},
	})
}

// SetupQueryRoutes 设置查询服务路由
func SetupQueryRoutes(router *gin.Engine) {
	queryService := &QueryService{}

	// 公开接口，不需要身份验证
	publicGroup := router.Group("/api/query")
	{
		publicGroup.GET("/trace", queryService.TraceProduct)
		publicGroup.GET("/verify", queryService.VerifyProduct)
	}

	// 需要身份验证的接口，监管方和消费者可访问
	authGroup := router.Group("/api/query")
	authGroup.Use(AuthMiddleware(), TypeAuthMiddleware(3))
	{
		// 如果有需要特殊权限的接口，可以放这里
	}
}
