package main

import (
	"bytes"
	"encoding/json"
	"go-user-api/handler"
	"go-user-api/logger"
	"go-user-api/model"
	"net/http" // HTTP 状态码
	"net/http/httptest"
	"strconv" // 数字转字符串
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert" // 断言库（判断结果对不对）
	"gorm.io/driver/sqlite"              // 内存数据库（测试专用）
	"gorm.io/gorm"
)

// 初始化测试环境：使用内存数据库，不影响真实数据
// 运行这个测试文件就是把下面每个测试函数运行一遍，且每个测试函数都会先运行setupTestEnv()这个函数
func setupTestEnv() (*gin.Engine, *gorm.DB) {
	// 👇 新增：在测试环境中初始化 zap 日志
	logger.InitLogger()

	// 使用内存 SQLite 做单元测试
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		panic("测试数据库连接失败")
	}

	// 自动迁移表结构
	err = db.AutoMigrate(&model.User{})
	if err != nil {
		panic("测试表迁移失败")
	}

	// 设置 Gin 为测试模式
	gin.SetMode(gin.TestMode)
	r := gin.Default()

	// 注册路由
	userHandler := handler.NewUserHandler(db)
	userGroup := r.Group("/users")
	{
		userGroup.POST("", userHandler.CreateUser)
		userGroup.GET("", userHandler.ListUsers)
		userGroup.GET("/:id", userHandler.GetUser)
		userGroup.PUT("/:id", userHandler.UpdateUser)
		userGroup.DELETE("/:id", userHandler.DeleteUser)
	}

	return r, db
}

// 测试：创建用户
func TestCreateUser(t *testing.T) {
	r, _ := setupTestEnv() //调用初始化虚拟化环境的函数

	// 构造请求体
	user := map[string]string{
		"username": "testuser",
		"email":    "test@example.com",
		"password": "123456",
	}
	jsonData, _ := json.Marshal(user)

	// 发送请求
	req := httptest.NewRequest(http.MethodPost, "/users", bytes.NewBuffer(jsonData))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	// 断言结果
	assert.Equal(t, http.StatusCreated, w.Code)
}

// 测试：获取用户列表
func TestListUsers(t *testing.T) {
	r, db := setupTestEnv()

	// 先插入一条测试数据
	db.Create(&model.User{Username: "user1", Email: "user1@test.com", Password: "123"})

	// 发送请求
	req := httptest.NewRequest(http.MethodGet, "/users", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	// 断言
	assert.Equal(t, http.StatusOK, w.Code)
}

// 测试：获取单个用户
func TestGetUser(t *testing.T) {
	r, db := setupTestEnv()
	user := model.User{Username: "getuser", Email: "get@test.com", Password: "123"}
	db.Create(&user)

	// 请求
	req := httptest.NewRequest(http.MethodGet, "/users/"+strconv.Itoa(int(user.ID)), nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

// 测试：更新用户
func TestUpdateUser(t *testing.T) {
	r, db := setupTestEnv()
	user := model.User{Username: "oldname", Email: "old@test.com", Password: "123"}
	db.Create(&user)

	// 更新数据
	updateData := map[string]string{
		"username": "newname",
		"email":    "new@test.com",
		"password": "654321",
	}
	jsonData, _ := json.Marshal(updateData)

	req := httptest.NewRequest(http.MethodPut, "/users/"+strconv.Itoa(int(user.ID)), bytes.NewBuffer(jsonData))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

// 测试：删除用户
func TestDeleteUser(t *testing.T) {
	r, db := setupTestEnv()
	user := model.User{Username: "deleteuser", Email: "del@test.com", Password: "123"}
	db.Create(&user)

	req := httptest.NewRequest(http.MethodDelete, "/users/"+strconv.Itoa(int(user.ID)), nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	// 删除成功返回 204
	assert.Equal(t, http.StatusNoContent, w.Code)
}
