package service

import (
	"back_Blockchain_cold_chain_traceability_system/api"
	"back_Blockchain_cold_chain_traceability_system/configs"
	"encoding/json"
	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
	"net/http"
	"strconv"
)

// AdminService 实现后台管理相关功能
type AdminService struct{}

// AdminUserList 获取用户列表
func (s *AdminService) AdminUserList(c *gin.Context) {
	userType := c.Query("user_type")
	auditStatus := c.Query("audit_status")
	keyword := c.Query("keyword")
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "10"))
	offset := (page - 1) * pageSize

	query := configs.DB.Model(&configs.User{})

	// 根据条件筛选
	if userType != "" {
		query = query.Where("user_type = ?", userType)
	}
	if auditStatus != "" {
		query = query.Where("audit_status = ?", auditStatus)
	}
	if keyword != "" {
		query = query.Where("username LIKE ? OR real_name LIKE ? OR company_name LIKE ?",
			"%"+keyword+"%", "%"+keyword+"%", "%"+keyword+"%")
	}

	var total int64
	query.Count(&total)

	var users []configs.User
	result := query.Order("created_at DESC").
		Offset(offset).
		Limit(pageSize).
		Find(&users)

	if result.Error != nil {
		c.JSON(http.StatusInternalServerError, api.Response{
			Code:    500,
			Message: "查询用户列表失败: " + result.Error.Error(),
		})
		return
	}

	// 不返回密码信息
	var responseUsers []gin.H
	for _, user := range users {
		responseUsers = append(responseUsers, gin.H{
			"id":           user.ID,
			"username":     user.Username,
			"real_name":    user.RealName,
			"address":      user.Address,
			"contact":      user.Contact,
			"user_type":    user.UserType,
			"company_name": user.CompanyName,
			"license_no":   user.LicenseNo,
			"audit_status": user.AuditStatus,
			"created_at":   user.CreatedAt,
		})
	}

	c.JSON(http.StatusOK, api.Response{
		Code:    200,
		Message: "获取用户列表成功",
		Data: gin.H{
			"total":     total,
			"page":      page,
			"page_size": pageSize,
			"users":     responseUsers,
		},
	})
}

// AdminProductList 获取产品列表
func (s *AdminService) AdminProductList(c *gin.Context) {
	status := c.Query("status")
	keyword := c.Query("keyword")
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "10"))
	offset := (page - 1) * pageSize

	query := configs.DB.Model(&configs.ProductInfo{})

	// 根据条件筛选
	if status != "" {
		query = query.Where("status = ?", status)
	}
	if keyword != "" {
		query = query.Where("name LIKE ? OR brand LIKE ? OR sku LIKE ?",
			"%"+keyword+"%", "%"+keyword+"%", "%"+keyword+"%")
	}

	var total int64
	query.Count(&total)

	var products []configs.ProductInfo
	result := query.Order("created_at DESC").
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
			"total":     total,
			"page":      page,
			"page_size": pageSize,
			"products":  products,
		},
	})
}

// AdminAuditUser 审核用户
func (s *AdminService) AdminAuditUser(c *gin.Context) {
	var req api.AuditRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, api.Response{
			Code:    400,
			Message: "请求参数错误: " + err.Error(),
		})
		return
	}

	var user configs.User
	result := configs.DB.First(&user, req.ID)
	if result.Error != nil {
		c.JSON(http.StatusNotFound, api.Response{
			Code:    404,
			Message: "用户不存在",
		})
		return
	}

	// 更新审核状态
	user.AuditStatus = req.Status
	user.AuditRemark = req.Remark
	result = configs.DB.Save(&user)

	if result.Error != nil {
		c.JSON(http.StatusInternalServerError, api.Response{
			Code:    500,
			Message: "更新用户审核状态失败: " + result.Error.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, api.Response{
		Code:    200,
		Message: "用户审核操作成功",
	})
}

// AdminAuditProduct 审核产品
func (s *AdminService) AdminAuditProduct(c *gin.Context) {
	var req api.AuditRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, api.Response{
			Code:    400,
			Message: "请求参数错误: " + err.Error(),
		})
		return
	}

	var product configs.ProductInfo
	result := configs.DB.First(&product, req.ID)
	if result.Error != nil {
		c.JSON(http.StatusNotFound, api.Response{
			Code:    404,
			Message: "产品不存在",
		})
		return
	}

	// 更新审核状态
	product.Status = req.Status
	product.AuditRemark = req.Remark
	result = configs.DB.Save(&product)

	if result.Error != nil {
		c.JSON(http.StatusInternalServerError, api.Response{
			Code:    500,
			Message: "更新产品审核状态失败: " + result.Error.Error(),
		})
		return
	}

	// 如果审核通过，记录到区块链
	if req.Status == 1 {
		blockchainService := &BlockchainService{}
		productData, _ := json.Marshal(product)
		blockchainService.AddToBlockchain(product.SKU, 1, string(productData))
	}

	c.JSON(http.StatusOK, api.Response{
		Code:    200,
		Message: "产品审核操作成功",
	})
}

// AdminAddUser 添加用户
func (s *AdminService) AdminAddUser(c *gin.Context) {
	var req api.UserRegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, api.Response{
			Code:    400,
			Message: "请求参数错误: " + err.Error(),
		})
		return
	}

	// 检查用户名是否已存在
	var existUser configs.User
	result := configs.DB.Where("username = ?", req.Username).First(&existUser)
	if result.RowsAffected > 0 {
		c.JSON(http.StatusBadRequest, api.Response{
			Code:    400,
			Message: "用户名已存在",
		})
		return
	}

	// 加密密码
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		c.JSON(http.StatusInternalServerError, api.Response{
			Code:    500,
			Message: "密码加密失败",
		})
		return
	}

	// 创建用户
	user := configs.User{
		Username:    req.Username,
		Password:    string(hashedPassword),
		RealName:    req.RealName,
		Address:     req.Address,
		Contact:     req.Contact,
		UserType:    req.UserType,
		CompanyName: req.CompanyName,
		LicenseNo:   req.LicenseNo,
		AuditStatus: 1, // 管理员添加直接审核通过
	}

	result = configs.DB.Create(&user)
	if result.Error != nil {
		c.JSON(http.StatusInternalServerError, api.Response{
			Code:    500,
			Message: "创建用户失败: " + result.Error.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, api.Response{
		Code:    200,
		Message: "添加用户成功",
		Data:    user.ID,
	})
}

// AdminDashboard 管理员仪表盘数据
func (s *AdminService) AdminDashboard(c *gin.Context) {
	// 统计用户数据
	var userStats struct {
		TotalUsers       int64 `json:"total_users"`
		PendingUsers     int64 `json:"pending_users"`
		FactoryCount     int64 `json:"factory_count"`
		DistributorCount int64 `json:"distributor_count"`
		ConsumerCount    int64 `json:"consumer_count"`
	}

	configs.DB.Model(&configs.User{}).Count(&userStats.TotalUsers)
	configs.DB.Model(&configs.User{}).Where("audit_status = 0").Count(&userStats.PendingUsers)
	configs.DB.Model(&configs.User{}).Where("user_type = 1").Count(&userStats.FactoryCount)
	configs.DB.Model(&configs.User{}).Where("user_type = 2").Count(&userStats.DistributorCount)
	configs.DB.Model(&configs.User{}).Where("user_type = 3").Count(&userStats.ConsumerCount)

	// 统计产品数据
	var productStats struct {
		TotalProducts   int64 `json:"total_products"`
		PendingProducts int64 `json:"pending_products"`
		ActiveProducts  int64 `json:"active_products"`
	}

	configs.DB.Model(&configs.ProductInfo{}).Count(&productStats.TotalProducts)
	configs.DB.Model(&configs.ProductInfo{}).Where("status = 0").Count(&productStats.PendingProducts)
	configs.DB.Model(&configs.ProductInfo{}).Where("status = 1").Count(&productStats.ActiveProducts)

	// 统计物流数据
	var logisticsCount int64
	configs.DB.Model(&configs.LogisticsRecord{}).Count(&logisticsCount)

	// 统计交接数据
	var transferCount int64
	configs.DB.Model(&configs.TransferRecord{}).Count(&transferCount)

	// 统计区块链数据
	var blockchainCount int64
	configs.DB.Model(&configs.BlockchainLog{}).Count(&blockchainCount)

	c.JSON(http.StatusOK, api.Response{
		Code:    200,
		Message: "获取仪表盘数据成功",
		Data: gin.H{
			"user_stats":       userStats,
			"product_stats":    productStats,
			"logistics_count":  logisticsCount,
			"transfer_count":   transferCount,
			"blockchain_count": blockchainCount,
		},
	})
}

// SetupAdminRoutes 设置管理员服务路由
func SetupAdminRoutes(router *gin.Engine) {
	adminService := &AdminService{}

	// 管理员接口
	adminGroup := router.Group("/api/admin")
	adminGroup.Use(AuthMiddleware(), TypeAuthMiddleware(4)) // 仅管理员可访问
	{
		adminGroup.GET("/users", adminService.AdminUserList)
		adminGroup.GET("/products", adminService.AdminProductList)
		adminGroup.POST("/user/audit", adminService.AdminAuditUser)
		adminGroup.POST("/product/audit", adminService.AdminAuditProduct)
		adminGroup.POST("/user/add", adminService.AdminAddUser)
		adminGroup.GET("/dashboard", adminService.AdminDashboard)
	}
}
