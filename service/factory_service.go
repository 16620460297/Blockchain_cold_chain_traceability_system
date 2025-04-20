package service

import (
	"Blockchain_cold_chain_traceability_system/api"
	"Blockchain_cold_chain_traceability_system/configs"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"time"
)

// FactoryService 实现厂家相关功能
type FactoryService struct{}

// AddProduct 添加冷冻品
func (s *FactoryService) AddProduct(c *gin.Context) {
	userID, _ := c.Get("userID")
	if userID == nil {
		c.JSON(http.StatusUnauthorized, api.Response{
			Code:    401,
			Message: "未登录",
		})
		return
	}

	var req api.AddProductRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, api.Response{
			Code:    400,
			Message: "请求参数错误: " + err.Error(),
		})
		return
	}

	// 解析日期
	productionDate, err := time.Parse("2006-01-02", req.ProductionDate)
	if err != nil {
		c.JSON(http.StatusBadRequest, api.Response{
			Code:    400,
			Message: "生产日期格式错误，应为YYYY-MM-DD",
		})
		return
	}

	expirationDate, err := time.Parse("2006-01-02", req.ExpirationDate)
	if err != nil {
		c.JSON(http.StatusBadRequest, api.Response{
			Code:    400,
			Message: "保质期格式错误，应为YYYY-MM-DD",
		})
		return
	}

	// 生成SKU码
	manufacturerID := userID.(uint)
	timeStr := time.Now().Format("20060102150405")
	uuidStr := uuid.New().String()[:8]
	sku := fmt.Sprintf("P%s%s%s", strconv.FormatUint(uint64(manufacturerID), 10), timeStr, uuidStr)

	// 保存图片
	imageURL := ""
	if req.ImageBase64 != "" {
		// 确保上传目录存在
		uploadDir := "./uploads/products"
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
		filename := sku + ".jpg"
		filePath := filepath.Join(uploadDir, filename)
		if err := os.WriteFile(filePath, imageData, 0644); err != nil {
			c.JSON(http.StatusInternalServerError, api.Response{
				Code:    500,
				Message: "保存图片失败",
			})
			return
		}

		// 设置图片URL
		imageURL = "/uploads/products/" + filename
	}

	// 创建产品记录
	product := configs.ProductInfo{
		SKU:              sku,
		Name:             req.Name,
		Brand:            req.Brand,
		Specification:    req.Specification,
		ProductionDate:   productionDate,
		ExpirationDate:   expirationDate,
		BatchNumber:      req.BatchNumber,
		ManufacturerID:   manufacturerID,
		MaterialSource:   req.MaterialSource,
		ProcessLocation:  req.ProcessLocation,
		ProcessMethod:    req.ProcessMethod,
		TransportTemp:    req.TransportTemp,
		StorageCondition: req.StorageCondition,
		SafetyTesting:    req.SafetyTesting,
		QualityRating:    req.QualityRating,
		ImageURL:         imageURL,
		Status:           0, // 默认待审核
	}

	result := configs.DB.Create(&product)
	if result.Error != nil {
		c.JSON(http.StatusInternalServerError, api.Response{
			Code:    500,
			Message: "创建产品失败: " + result.Error.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, api.Response{
		Code:    200,
		Message: "添加产品成功，请等待审核",
		Data: gin.H{
			"product_id": product.ID,
			"sku":        product.SKU,
		},
	})
}

// GetProductList 获取厂家的产品列表
func (s *FactoryService) GetProductList(c *gin.Context) {
	userID, _ := c.Get("userID")
	if userID == nil {
		c.JSON(http.StatusUnauthorized, api.Response{
			Code:    401,
			Message: "未登录",
		})
		return
	}

	// 分页参数
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "10"))
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 10
	}
	offset := (page - 1) * pageSize

	// 查询过滤
	status := c.Query("status")
	statusFilter := ""
	if status != "" {
		statusFilter = "status = " + status
	}

	// 查询产品列表
	var products []configs.ProductInfo
	query := configs.DB.Where("manufacturer_id = ?", userID)
	if statusFilter != "" {
		query = query.Where(statusFilter)
	}

	var total int64
	query.Model(&configs.ProductInfo{}).Count(&total)

	result := query.Order("created_at DESC").Offset(offset).Limit(pageSize).Find(&products)
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

// ConfirmTransfer 确认产品交接
func (s *FactoryService) ConfirmTransfer(c *gin.Context) {
	userID, _ := c.Get("userID")
	if userID == nil {
		c.JSON(http.StatusUnauthorized, api.Response{
			Code:    401,
			Message: "未登录",
		})
		return
	}

	var req api.TransferConfirmRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, api.Response{
			Code:    400,
			Message: "请求参数错误: " + err.Error(),
		})
		return
	}

	// 检查产品是否存在且是当前用户的
	var product configs.ProductInfo
	result := configs.DB.Where("sku = ? AND manufacturer_id = ?", req.ProductSKU, userID).First(&product)
	if result.Error != nil {
		c.JSON(http.StatusNotFound, api.Response{
			Code:    404,
			Message: "产品不存在或不属于当前用户",
		})
		return
	}

	// 续上段代码
	// 检查产品是否已发布
	if product.Status != 1 {
		c.JSON(http.StatusBadRequest, api.Response{
			Code:    400,
			Message: "产品尚未审核通过，无法确认交接",
		})
		return
	}

	// 检查目标用户是否存在且是经销商
	var toUser configs.User
	result = configs.DB.Where("id = ? AND user_type = 2", req.ToUserID).First(&toUser)
	if result.Error != nil {
		c.JSON(http.StatusNotFound, api.Response{
			Code:    404,
			Message: "目标用户不存在或不是经销商",
		})
		return
	}

	// 创建交接记录
	transfer := configs.TransferRecord{
		ProductSKU: req.ProductSKU,
		FromUserID: userID.(uint),
		ToUserID:   req.ToUserID,
		Remarks:    req.Remarks,
		Status:     1, // 厂家直接确认
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

// UpdateProduct 更新产品信息
func (s *FactoryService) UpdateProduct(c *gin.Context) {
	userID, _ := c.Get("userID")
	productID := c.Param("id")

	var product configs.ProductInfo
	result := configs.DB.Where("id = ? AND manufacturer_id = ?", productID, userID).First(&product)
	if result.Error != nil {
		c.JSON(http.StatusNotFound, api.Response{
			Code:    404,
			Message: "产品不存在或不属于当前用户",
		})
		return
	}

	// 只有待审核状态的产品才能更新
	if product.Status != 0 {
		c.JSON(http.StatusBadRequest, api.Response{
			Code:    400,
			Message: "只有待审核的产品才能更新",
		})
		return
	}

	var updateData api.AddProductRequest
	if err := c.ShouldBindJSON(&updateData); err != nil {
		c.JSON(http.StatusBadRequest, api.Response{
			Code:    400,
			Message: "请求参数错误: " + err.Error(),
		})
		return
	}

	// 更新基本信息
	if updateData.Name != "" {
		product.Name = updateData.Name
	}
	if updateData.Brand != "" {
		product.Brand = updateData.Brand
	}
	if updateData.Specification != "" {
		product.Specification = updateData.Specification
	}
	if updateData.BatchNumber != "" {
		product.BatchNumber = updateData.BatchNumber
	}
	if updateData.MaterialSource != "" {
		product.MaterialSource = updateData.MaterialSource
	}
	if updateData.ProcessLocation != "" {
		product.ProcessLocation = updateData.ProcessLocation
	}
	if updateData.ProcessMethod != "" {
		product.ProcessMethod = updateData.ProcessMethod
	}
	if updateData.TransportTemp != 0 {
		product.TransportTemp = updateData.TransportTemp
	}
	if updateData.StorageCondition != "" {
		product.StorageCondition = updateData.StorageCondition
	}
	if updateData.SafetyTesting != "" {
		product.SafetyTesting = updateData.SafetyTesting
	}
	if updateData.QualityRating != "" {
		product.QualityRating = updateData.QualityRating
	}

	// 处理日期
	if updateData.ProductionDate != "" {
		productionDate, err := time.Parse("2006-01-02", updateData.ProductionDate)
		if err == nil {
			product.ProductionDate = productionDate
		}
	}
	if updateData.ExpirationDate != "" {
		expirationDate, err := time.Parse("2006-01-02", updateData.ExpirationDate)
		if err == nil {
			product.ExpirationDate = expirationDate
		}
	}

	// 处理图片更新
	if updateData.ImageBase64 != "" {
		// 确保上传目录存在
		uploadDir := "./uploads/products"
		if err := os.MkdirAll(uploadDir, 0755); err != nil {
			c.JSON(http.StatusInternalServerError, api.Response{
				Code:    500,
				Message: "创建上传目录失败",
			})
			return
		}

		// 删除旧图片（如果存在）
		if product.ImageURL != "" {
			oldImagePath := "." + product.ImageURL
			if _, err := os.Stat(oldImagePath); err == nil {
				os.Remove(oldImagePath)
			}
		}

		// 解码并保存新图片
		imageData, err := base64.StdEncoding.DecodeString(updateData.ImageBase64)
		if err != nil {
			c.JSON(http.StatusBadRequest, api.Response{
				Code:    400,
				Message: "图片格式错误",
			})
			return
		}

		filename := product.SKU + "_" + strconv.FormatInt(time.Now().Unix(), 10) + ".jpg"
		filePath := filepath.Join(uploadDir, filename)
		if err := os.WriteFile(filePath, imageData, 0644); err != nil {
			c.JSON(http.StatusInternalServerError, api.Response{
				Code:    500,
				Message: "保存图片失败",
			})
			return
		}

		product.ImageURL = "/uploads/products/" + filename
	}

	// 保存更新
	result = configs.DB.Save(&product)
	if result.Error != nil {
		c.JSON(http.StatusInternalServerError, api.Response{
			Code:    500,
			Message: "更新产品失败: " + result.Error.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, api.Response{
		Code:    200,
		Message: "更新产品成功",
	})
}

// GetPendingTransfers 获取待确认的交接记录
func (s *FactoryService) GetPendingTransfers(c *gin.Context) {
	userID, _ := c.Get("userID")

	// 分页参数
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "10"))
	offset := (page - 1) * pageSize

	var transfers []struct {
		configs.TransferRecord
		ProductName  string `json:"product_name"`
		FromUsername string `json:"from_username"`
		ToUsername   string `json:"to_username"`
	}

	// 查询待确认的交接记录
	query := configs.DB.Table("transfer_records").
		Select("transfer_records.*, p.name as product_name, u1.username as from_username, u2.username as to_username").
		Joins("JOIN product_infos p ON transfer_records.product_sku = p.sku").
		Joins("JOIN users u1 ON transfer_records.from_user_id = u1.id").
		Joins("JOIN users u2 ON transfer_records.to_user_id = u2.id").
		Where("(from_user_id = ? OR to_user_id = ?) AND status = 0", userID, userID)

	var total int64
	query.Count(&total)

	result := query.Order("transfer_records.created_at DESC").
		Offset(offset).
		Limit(pageSize).
		Find(&transfers)

	if result.Error != nil {
		c.JSON(http.StatusInternalServerError, api.Response{
			Code:    500,
			Message: "查询交接记录失败: " + result.Error.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, api.Response{
		Code:    200,
		Message: "获取待确认交接记录成功",
		Data: gin.H{
			"total":     total,
			"page":      page,
			"page_size": pageSize,
			"transfers": transfers,
		},
	})
}

// SetupFactoryRoutes 设置厂家服务路由
func SetupFactoryRoutes(router *gin.Engine) {
	factoryService := &FactoryService{}

	factoryGroup := router.Group("/api/factory")
	factoryGroup.Use(AuthMiddleware(), TypeAuthMiddleware(1)) // 仅厂家可访问
	{
		factoryGroup.POST("/product", factoryService.AddProduct)
		factoryGroup.GET("/products", factoryService.GetProductList)
		factoryGroup.PUT("/product/:id", factoryService.UpdateProduct)
		factoryGroup.POST("/transfer/confirm", factoryService.ConfirmTransfer)
		factoryGroup.GET("/transfers/pending", factoryService.GetPendingTransfers)
	}
}
