package main

import (
	"go-user-api/handler"
	"go-user-api/logger"
	"go-user-api/model" //用户结构体
	"os"

	"github.com/gin-gonic/gin" // Web 框架，写 API 用
	"github.com/joho/godotenv" // 读取 .env 配置文件
	"go.uber.org/zap"          //日志文件
	"gorm.io/driver/mysql"
	"gorm.io/gorm" //操作数据库
)

func main() {
	// 初始化日志
	logger.InitLogger()

	// 加载.env里面的环境变量到当前系统变量中，err！=nil意思是问题不为空，也就是有问题，即出错了
	if err := godotenv.Load(); err != nil {
		logger.Log.Warn("No .env file found, using system environment variables") // 替换 log.Println
	}

	// 拼接数据库连接字符串
	dsn := os.Getenv("DB_USER") + ":" + os.Getenv("DB_PASSWORD") + "@tcp(" + os.Getenv("DB_HOST") + ":" + os.Getenv("DB_PORT") + ")/" + os.Getenv("DB_NAME") + "?charset=utf8mb4&parseTime=True&loc=Local"

	// 连接 MySQL
	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		logger.Log.Fatal("Failed to connect to database", zap.Error(err)) // 替换 log.Fatal
	}

	// 自动迁移表结构
	if err := db.AutoMigrate(&model.User{}); err != nil {
		logger.Log.Fatal("Failed to migrate database", zap.Error(err)) // 替换 log.Fatal
	}

	// 初始化 Gin 路由，创建一个 Web 服务，用来接收 HTTP 请求
	r := gin.Default()

	// 注册用户路由
	userHandler := handler.NewUserHandler(db)
	userGroup := r.Group("/users")
	{
		userGroup.POST("", userHandler.CreateUser)
		userGroup.GET("", userHandler.ListUsers)
		userGroup.GET("/:id", userHandler.GetUser)
		userGroup.PUT("/:id", userHandler.UpdateUser)
		userGroup.DELETE("/:id", userHandler.DeleteUser)
	}

	// 启动服务
	logger.Log.Info("Server starting on :8080...") // 替换 log.Println
	if err := r.Run(":8080"); err != nil {
		logger.Log.Fatal("Failed to start server", zap.Error(err)) // 替换 log.Fatal
	}
}
