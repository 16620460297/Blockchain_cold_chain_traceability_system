package service

import (
	"back_Blockchain_cold_chain_traceability_system/api"
	"back_Blockchain_cold_chain_traceability_system/configs"
	"encoding/base64"
	"encoding/json"
	"github.com/gin-gonic/gin"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"time"
)

// SalerService 实现经销商相关功能
type SalerService struct{}

// SearchFactories 搜索厂家
func (s *SalerService) SearchFactories(c *gin.Context) {
	keyword := c.Query("keyword")
	if keyword == "" {
		c.JSON(http.StatusBadRequest, api.Response{
			Code:    400,
			Message: "搜索关键词不能为空",
		})
		return
	}

	// 查询已审核通过的厂家
	var factories []configs.User
	result := configs.DB.Where("(username LIKE ? OR real_name LIKE ? OR company_name LIKE ?) AND user_type = 1 AND audit_status = 1",
		"%"+keyword+"%", "%"+keyword+"%", "%"+keyword+"%").
		Select("id, username, real_name, company_name, address, contact").
		Limit(10).
		Find(&factories)

	if result.Error != nil {
		c.JSON(http.StatusInternalServerError, api.Response{
			Code:    500,
			Message: "搜索厂家失败: " + result.Error.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, api.Response{
		Code:    200,
		Message: "搜索厂家成功",
		Data:    factories,
	})
}

// SearchDistributors 搜索经销商
func (s *SalerService) SearchDistributors(c *gin.Context) {
	keyword := c.Query("keyword")
	if keyword == "" {
		c.JSON(http.StatusBadRequest, api.Response{
			Code:    400,
			Message: "搜索关键词不能为空",
		})
		return
	}

	// 查询已审核通过的经销商
	var distributors []configs.User
	result := configs.DB.Where("(username LIKE ? OR real_name LIKE ? OR company_name LIKE ?) AND user_type = 2 AND audit_status = 1",
		"%"+keyword+"%", "%"+keyword+"%", "%"+keyword+"%").
		Select("id, username, real_name, company_name, address, contact").
		Limit(10).
		Find(&distributors)

	if result.Error != nil {
		c.JSON(http.StatusInternalServerError, api.Response{
			Code:    500,
			Message: "搜索经销商失败: " + result.Error.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, api.Response{
		Code:    200,
		Message: "搜索经销商成功",
		Data:    distributors,
	})
}

// SearchProducts 搜索产品
func (s *SalerService) SearchProducts(c *gin.Context) {
	keyword := c.Query("keyword")
	factoryID := c.Query("factory_id")

	// 至少需要一个搜索条件
	if keyword == "" && factoryID == "" {
		c.JSON(http.StatusBadRequest, api.Response{
			Code:    400,
			Message: "请提供搜索关键词或厂家ID",
		})
		return
	}

	// 构建查询
	query := configs.DB.Model(&configs.ProductInfo{}).Where("status = 1") // 只查询已上架的产品

	if keyword != "" {
		query = query.Where("name LIKE ? OR brand LIKE ? OR sku LIKE ?",
			"%"+keyword+"%", "%"+keyword+"%", "%"+keyword+"%")
	}

	if factoryID != "" {
		query = query.Where("manufacturer_id = ?", factoryID)
	}

	// 执行查询
	var products []configs.ProductInfo
	result := query.Limit(10).Find(&products)

	if result.Error != nil {
		c.JSON(http.StatusInternalServerError, api.Response{
			Code:    500,
			Message: "搜索产品失败: " + result.Error.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, api.Response{
		Code:    200,
		Message: "搜索产品成功",
		Data:    products,
	})
}

// AddLogistics 添加物流信息
func (s *SalerService) AddLogistics(c *gin.Context) {
	userID, _ := c.Get("userID")
	userType, _ := c.Get("userType")

	var req api.LogisticsInfo
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, api.Response{
			Code:    400,
			Message: "请求参数错误: " + err.Error(),
		})
		return
	}

	// 检查产品是否存在
	var product configs.ProductInfo
	result := configs.DB.Where("sku = ? AND status = 1", req.ProductSKU).First(&product)
	if result.Error != nil {
		c.JSON(http.StatusNotFound, api.Response{
			Code:    404,
			Message: "产品不存在或未上架",
		})
		return
	}

	// 处理图片
	imageURL := ""
	if req.ImageBase64 != "" {
		// 确保上传目录存在
		uploadDir := "./uploads/logistics"
		if err := os.MkdirAll(uploadDir, 0755); err != nil {
			c.JSON(http.StatusInternalServerError, api.Response{
				Code:    500,
				Message: "创建上传目录失败",
			})
			return
		}

		// 解码Base64图片
		imageData, err := base64.StdEncoding.DecodeString(req.ImageBase64)
		if err != nil {
			c.JSON(http.StatusBadRequest, api.Response{
				Code:    400,
				Message: "图片格式错误",
			})
			return
		}

		// 保存图片
		filename := req.ProductSKU + "_" + strconv.FormatInt(time.Now().Unix(), 10) + ".jpg"
		filePath := filepath.Join(uploadDir, filename)
		if err := os.WriteFile(filePath, imageData, 0644); err != nil {
			c.JSON(http.StatusInternalServerError, api.Response{
				Code:    500,
				Message: "保存图片失败",
			})
			return
		}

		imageURL = "/uploads/logistics/" + filename
	}

	// 创建物流记录
	logistics := configs.LogisticsRecord{
		ProductSKU:        req.ProductSKU,
		TrackingNo:        req.TrackingNo,
		WarehouseLocation: req.WarehouseLocation,
		Temperature:       req.Temperature,
		Humidity:          req.Humidity,
		ImageURL:          imageURL,
		OperatorID:        userID.(uint),
		OperatorType:      userType.(int),
	}

	result = configs.DB.Create(&logistics)
	if result.Error != nil {
		c.JSON(http.StatusInternalServerError, api.Response{
			Code:    500,
			Message: "创建物流记录失败: " + result.Error.Error(),
		})
		return
	}

	// 记录到区块链
	blockchainService := &BlockchainService{}
	logisticsData, _ := json.Marshal(logistics)
	blockchainService.AddToBlockchain(req.ProductSKU, 2, string(logisticsData))

	c.JSON(http.StatusOK, api.Response{
		Code:    200,
		Message: "添加物流信息成功",
		Data:    logistics.ID,
	})
}

// ConfirmTransfer 确认产品交接
func (s *SalerService) ConfirmTransfer(c *gin.Context) {
	userID, _ := c.Get("userID")

	var req api.TransferConfirmRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, api.Response{
			Code:    400,
			Message: "请求参数错误: " + err.Error(),
		})
		return
	}

	// 检查产品是否存在
	var product configs.ProductInfo
	result := configs.DB.Where("sku = ? AND status = 1", req.ProductSKU).First(&product)
	if result.Error != nil {
		c.JSON(http.StatusNotFound, api.Response{
			Code:    404,
			Message: "产品不存在或未上架",
		})
		return
	}

	// 检查目标用户是否存在且是经销商
	var toUser configs.User
	result = configs.DB.Where("id = ? AND user_type = 2 AND audit_status = 1", req.ToUserID).First(&toUser)
	if result.Error != nil {
		c.JSON(http.StatusNotFound, api.Response{
			Code:    404,
			Message: "目标用户不存在或不是已审核的经销商",
		})
		return
	}

	// 创建交接记录
	transfer := configs.TransferRecord{
		ProductSKU: req.ProductSKU,
		FromUserID: userID.(uint),
		ToUserID:   req.ToUserID,
		Remarks:    req.Remarks,
		Status:     1, // 直接确认
	}

	result = configs.DB.Create(&transfer)
	if result.Error != nil {
		c.JSON(http.StatusInternalServerError, api.Response{
			Code:    500,
			Message: "创建交接记录失败: " + result.Error.Error(),
		})
		return
	}

	// 记录到区块链
	blockchainService := &BlockchainService{}
	transferData, _ := json.Marshal(transfer)
	blockchainService.AddToBlockchain(req.ProductSKU, 3, string(transferData))

	c.JSON(http.StatusOK, api.Response{
		Code:    200,
		Message: "确认交接成功",
		Data:    transfer.ID,
	})
}

// GetProductList 获取经销商关联的产品列表
func (s *SalerService) GetProductList(c *gin.Context) {
	userID, _ := c.Get("userID")

	// 分页参数
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "10"))
	offset := (page - 1) * pageSize

	// 查询经销商接收的所有产品
	var productSKUs []string
	result := configs.DB.Model(&configs.TransferRecord{}).
		Where("to_user_id = ? AND status = 1", userID).
		Distinct("product_sku").
		Pluck("product_sku", &productSKUs)

	if result.Error != nil {
		c.JSON(http.StatusInternalServerError, api.Response{
			Code:    500,
			Message: "查询产品失败: " + result.Error.Error(),
		})
		return
	}

	if len(productSKUs) == 0 {
		c.JSON(http.StatusOK, api.Response{
			Code:    200,
			Message: "获取产品列表成功",
			Data: gin.H{
				"total":    0,
				"page":     page,
				"pageSize": pageSize,
				"products": []interface{}{},
			},
		})
		return
	}

	// 查询这些产品的详细信息
	var products []configs.ProductInfo
	var total int64
	result = configs.DB.Model(&configs.ProductInfo{}).
		Where("sku IN ? AND status = 1", productSKUs).
		Count(&total)

	result = configs.DB.Where("sku IN ? AND status = 1", productSKUs).
		Order("created_at DESC").
		Offset(offset).
		Limit(pageSize).
		Find(&products)

	if result.Error != nil {
		c.JSON(http.StatusInternalServerError, api.Response{
			Code:    500,
			Message: "查询产品列表失败: " + result.Error.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, api.Response{
		Code:    200,
		Message: "获取产品列表成功",
		Data: gin.H{
			"total":    total,
			"page":     page,
			"pageSize": pageSize,
			"products": products,
		},
	})
}

// GetLogisticsList 获取产品的物流记录
func (s *SalerService) GetLogisticsList(c *gin.Context) {
	productSKU := c.Query("sku")
	if productSKU == "" {
		c.JSON(http.StatusBadRequest, api.Response{
			Code:    400,
			Message: "请提供产品SKU",
		})
		return
	}

	var logistics []struct {
		configs.LogisticsRecord
		OperatorName string `json:"operator_name"`
	}

	result := configs.DB.Table("logistics_records").
		Select("logistics_records.*, users.real_name as operator_name").
		Joins("JOIN users ON logistics_records.operator_id = users.id").
		Where("logistics_records.product_sku = ?", productSKU).
		Order("logistics_records.created_at DESC").
		Find(&logistics)

	if result.Error != nil {
		c.JSON(http.StatusInternalServerError, api.Response{
			Code:    500,
			Message: "查询物流记录失败: " + result.Error.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, api.Response{
		Code:    200,
		Message: "获取物流记录成功",
		Data:    logistics,
	})
}

// SetupSalerRoutes 设置经销商服务路由
func SetupSalerRoutes(router *gin.Engine) {
	salerService := &SalerService{}

	salerGroup := router.Group("/api/saler")
	salerGroup.Use(AuthMiddleware(), TypeAuthMiddleware(2)) // 仅经销商可访问
	{
		salerGroup.GET("/factory/search", salerService.SearchFactories)
		salerGroup.GET("/distributor/search", salerService.SearchDistributors)
		salerGroup.GET("/product/search", salerService.SearchProducts)
		salerGroup.POST("/logistics", salerService.AddLogistics)
		salerGroup.POST("/transfer/confirm", salerService.ConfirmTransfer)
		salerGroup.GET("/products", salerService.GetProductList)
		salerGroup.GET("/logistics/list", salerService.GetLogisticsList)
	}
}
