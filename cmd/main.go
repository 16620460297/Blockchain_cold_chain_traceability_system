package main

import (
	"back_Blockchain_cold_chain_traceability_system/configs"
	"back_Blockchain_cold_chain_traceability_system/service"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
	"log"
	"os"
	"time"
)

func main() {
	// 初始化数据库
	err := configs.InitMySQL()
	if err != nil {
		log.Fatalf("未能初始化 MySQL: %v", err)
	} else {
		log.Println("MySQL 初始化成功")
	}

	// 初始化Redis
	err = configs.InitRedis()
	if err != nil {
		log.Fatalf("未能初始化 Redis。: %v", err)
	} else {
		log.Println("Redis 初始化成功")
	}

	// 初始化上传目录
	ensureDir("./uploads/products")
	ensureDir("./uploads/logistics")

	// 设置Gin模式
	gin.SetMode(gin.ReleaseMode)
	if os.Getenv("GIN_MODE") == "debug" {
		gin.SetMode(gin.DebugMode)
	}

	// 创建Gin实例
	r := gin.Default()

	// CORS设置
	r.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"*"},
		AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Accept", "Authorization"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}))

	// 静态文件服务
	r.Static("/uploads", "./uploads")

	// 注册路由
	service.SetupUserRoutes(r)
	service.SetupFactoryRoutes(r)
	service.SetupSalerRoutes(r)
	service.SetupQueryRoutes(r)
	service.SetupBlockchainRoutes(r)
	service.SetupAdminRoutes(r)

	// 初始化管理员账户
	initAdminUser()

	// 获取端口配置
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	// 启动服务器
	log.Printf("Server starting on port %s...", port)
	err = r.Run("0.0.0.0:" + port)
	if err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}

// 初始化管理员账户
func initAdminUser() {
	var count int64
	configs.DB.Model(&configs.User{}).Where("user_type = 4").Count(&count)
	if count > 0 {
		return
	}

	password, _ := bcrypt.GenerateFromPassword([]byte("admin123"), bcrypt.DefaultCost)
	admin := configs.User{
		Username:    "admin",
		Password:    string(password),
		RealName:    "系统管理员",
		Address:     "系统",
		Contact:     "admin@system.com",
		UserType:    4,
		AuditStatus: 1,
	}

	result := configs.DB.Create(&admin)
	if result.Error != nil {
		log.Printf("Failed to create admin user: %v", result.Error)
	} else {
		log.Println("Admin user created successfully")
	}
}

// 确保目录存在
func ensureDir(dirPath string) {
	if _, err := os.Stat(dirPath); os.IsNotExist(err) {
		err = os.MkdirAll(dirPath, 0755)
		if err != nil {
			log.Printf("Failed to create directory %s: %v", dirPath, err)
		}
	}
}
