package service

import (
	"Blockchain_cold_chain_traceability_system/api"
	"Blockchain_cold_chain_traceability_system/configs"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
	"net/http"
	"time"
)

// JWT密钥
var jwtKey = []byte("cold_chain_secret_key")

// 自定义JWT声明结构
type Claims struct {
	UserID   uint
	Username string
	UserType int
	jwt.RegisteredClaims
}

// UserService 实现用户相关功能
type UserService struct{}

// Register 用户注册
func (s *UserService) Register(c *gin.Context) {
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
		AuditStatus: 0, // 默认未审核
	}

	// 如果是普通消费者，自动审核通过
	if req.UserType == 3 {
		user.AuditStatus = 1
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
		Message: "注册成功，请等待审核",
		Data:    user.ID,
	})
}

// Login 用户登录
func (s *UserService) Login(c *gin.Context) {
	var req api.UserLoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, api.Response{
			Code:    400,
			Message: "请求参数错误: " + err.Error(),
		})
		return
	}

	// 查找用户
	var user configs.User
	result := configs.DB.Where("username = ?", req.Username).First(&user)
	if result.Error != nil {
		c.JSON(http.StatusUnauthorized, api.Response{
			Code:    401,
			Message: "用户名或密码错误",
		})
		return
	}

	// 验证密码
	err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.Password))
	if err != nil {
		c.JSON(http.StatusUnauthorized, api.Response{
			Code:    401,
			Message: "用户名或密码错误",
		})
		return
	}

	// 生成JWT Token
	expirationTime := time.Now().Add(24 * time.Hour)
	claims := &Claims{
		UserID:   user.ID,
		Username: user.Username,
		UserType: user.UserType,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expirationTime),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString(jwtKey)
	if err != nil {
		c.JSON(http.StatusInternalServerError, api.Response{
			Code:    500,
			Message: "生成token失败",
		})
		return
	}

	// 存储到Redis
	err = configs.StoreJWT(user.ID, tokenString, 24*time.Hour)
	if err != nil {
		c.JSON(http.StatusInternalServerError, api.Response{
			Code:    500,
			Message: "存储token失败",
		})
		return
	}

	c.JSON(http.StatusOK, api.Response{
		Code:    200,
		Message: "登录成功",
		Data: api.UserLoginResponse{
			Token:       tokenString,
			UserID:      user.ID,
			Username:    user.Username,
			UserType:    user.UserType,
			RealName:    user.RealName,
			AuditStatus: user.AuditStatus,
		},
	})
}

// Logout 用户退出登录
func (s *UserService) Logout(c *gin.Context) {
	userID, _ := c.Get("userID")
	if userID == nil {
		c.JSON(http.StatusUnauthorized, api.Response{
			Code:    401,
			Message: "未登录",
		})
		return
	}

	// 从Redis删除token
	err := configs.DeleteJWT(userID.(uint))
	if err != nil {
		c.JSON(http.StatusInternalServerError, api.Response{
			Code:    500,
			Message: "退出登录失败",
		})
		return
	}

	c.JSON(http.StatusOK, api.Response{
		Code:    200,
		Message: "退出登录成功",
	})
}

// GetUserInfo 获取用户信息
func (s *UserService) GetUserInfo(c *gin.Context) {
	userID, _ := c.Get("userID")
	if userID == nil {
		c.JSON(http.StatusUnauthorized, api.Response{
			Code:    401,
			Message: "未登录",
		})
		return
	}

	var user configs.User
	result := configs.DB.Where("id = ?", userID).First(&user)
	if result.Error != nil {
		c.JSON(http.StatusNotFound, api.Response{
			Code:    404,
			Message: "用户不存在",
		})
		return
	}

	// 不返回密码等敏感信息
	c.JSON(http.StatusOK, api.Response{
		Code:    200,
		Message: "获取用户信息成功",
		Data: gin.H{
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
		},
	})
}

// UpdateUserInfo 更新用户信息
func (s *UserService) UpdateUserInfo(c *gin.Context) {
	userID, _ := c.Get("userID")
	if userID == nil {
		c.JSON(http.StatusUnauthorized, api.Response{
			Code:    401,
			Message: "未登录",
		})
		return
	}

	var updateData struct {
		RealName    string `json:"real_name"`
		Address     string `json:"address"`
		Contact     string `json:"contact"`
		CompanyName string `json:"company_name"`
		LicenseNo   string `json:"license_no"`
	}

	if err := c.ShouldBindJSON(&updateData); err != nil {
		c.JSON(http.StatusBadRequest, api.Response{
			Code:    400,
			Message: "请求参数错误: " + err.Error(),
		})
		return
	}

	var user configs.User
	result := configs.DB.Where("id = ?", userID).First(&user)
	if result.Error != nil {
		c.JSON(http.StatusNotFound, api.Response{
			Code:    404,
			Message: "用户不存在",
		})
		return
	}

	// 更新字段
	if updateData.RealName != "" {
		user.RealName = updateData.RealName
	}
	if updateData.Address != "" {
		user.Address = updateData.Address
	}
	if updateData.Contact != "" {
		user.Contact = updateData.Contact
	}
	if updateData.CompanyName != "" {
		user.CompanyName = updateData.CompanyName
	}
	if updateData.LicenseNo != "" {
		user.LicenseNo = updateData.LicenseNo
	}

	result = configs.DB.Save(&user)
	if result.Error != nil {
		c.JSON(http.StatusInternalServerError, api.Response{
			Code:    500,
			Message: "更新用户信息失败: " + result.Error.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, api.Response{
		Code:    200,
		Message: "更新用户信息成功",
	})
}

// AuthMiddleware JWT认证中间件
func AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		tokenString := c.GetHeader("Authorization")
		if tokenString == "" {
			c.JSON(http.StatusUnauthorized, api.Response{
				Code:    401,
				Message: "未提供认证令牌",
			})
			c.Abort()
			return
		}

		// 移除"Bearer "前缀（如果有）
		if len(tokenString) > 7 && tokenString[:7] == "Bearer " {
			tokenString = tokenString[7:]
		}

		// 解析JWT
		claims := &Claims{}
		token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
			return jwtKey, nil
		})

		if err != nil || !token.Valid {
			c.JSON(http.StatusUnauthorized, api.Response{
				Code:    401,
				Message: "无效的认证令牌",
			})
			c.Abort()
			return
		}

		// 从Redis验证token
		storedToken, err := configs.GetJWT(claims.UserID)
		if err != nil || storedToken != tokenString {
			c.JSON(http.StatusUnauthorized, api.Response{
				Code:    401,
				Message: "无效的认证令牌或已过期",
			})
			c.Abort()
			return
		}

		// 设置用户信息到上下文
		c.Set("userID", claims.UserID)
		c.Set("username", claims.Username)
		c.Set("userType", claims.UserType)

		c.Next()
	}
}

// TypeAuthMiddleware 用户类型认证中间件
func TypeAuthMiddleware(allowedTypes ...int) gin.HandlerFunc {
	return func(c *gin.Context) {
		userType, exists := c.Get("userType")
		if !exists {
			c.JSON(http.StatusUnauthorized, api.Response{
				Code:    401,
				Message: "认证失败",
			})
			c.Abort()
			return
		}

		allowed := false
		for _, t := range allowedTypes {
			if userType.(int) == t {
				allowed = true
				break
			}
		}

		if !allowed {
			c.JSON(http.StatusForbidden, api.Response{
				Code:    403,
				Message: "权限不足",
			})
			c.Abort()
			return
		}

		// 检查是否已通过审核
		userID, _ := c.Get("userID")
		var user configs.User
		result := configs.DB.Select("audit_status").Where("id = ?", userID).First(&user)
		if result.Error == nil && user.AuditStatus != 1 {
			c.JSON(http.StatusForbidden, api.Response{
				Code:    403,
				Message: "账号尚未通过审核",
			})
			c.Abort()
			return
		}

		c.Next()
	}
}

// SetupUserRoutes 设置用户服务路由
func SetupUserRoutes(router *gin.Engine) {
	userService := &UserService{}

	userGroup := router.Group("/api/user")
	{
		userGroup.POST("/register", userService.Register)
		userGroup.POST("/login", userService.Login)

		// 需要认证的路由
		authGroup := userGroup.Group("/")
		authGroup.Use(AuthMiddleware())
		{
			authGroup.POST("/logout", userService.Logout)
			authGroup.GET("/info", userService.GetUserInfo)
			authGroup.PUT("/info", userService.UpdateUserInfo)
		}
	}
}
