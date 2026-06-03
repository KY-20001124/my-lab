package handler

import (
	"go-user-api/logger"
	"go-user-api/model"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type UserHandler struct {
	db *gorm.DB
}

func NewUserHandler(db *gorm.DB) *UserHandler {
	return &UserHandler{db: db}
}

// CreateUser 创建用户
func (h *UserHandler) CreateUser(c *gin.Context) {
	var user model.User
	//把c里面的上下文填充到空白结构体user里面
	//失败返回400
	if err := c.ShouldBindJSON(&user); err != nil {
		logger.Log.Warn("Invalid request body", zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	//把user数据写入数据库，失败返回500
	if err := h.db.Create(&user).Error; err != nil {
		logger.Log.Error("Failed to create user", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	//返回成功结果201
	logger.Log.Info("User created", zap.String("username", user.Username))
	c.JSON(http.StatusCreated, user)
}

// ListUsers 列出所有用户
func (h *UserHandler) ListUsers(c *gin.Context) {
	var users []model.User
	//如果没有用户的话会返回空值，并返回状态码200
	if err := h.db.Find(&users).Error; err != nil {
		logger.Log.Error("Failed to list users", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	logger.Log.Info("Listed all users", zap.Int("count", len(users)))
	c.JSON(http.StatusOK, users)
}

// GetUser 获取单个用户
func (h *UserHandler) GetUser(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	var user model.User
	if err := h.db.First(&user, id).Error; err != nil {
		logger.Log.Warn("User not found", zap.Int("id", id))
		c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
		return
	}
	logger.Log.Info("Fetched user", zap.Int("id", id))
	c.JSON(http.StatusOK, user)
}

// UpdateUser 更新用户
func (h *UserHandler) UpdateUser(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	var user model.User
	if err := h.db.First(&user, id).Error; err != nil {
		logger.Log.Warn("User not found for update", zap.Int("id", id))
		c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
		return
	}
	if err := c.ShouldBindJSON(&user); err != nil {
		logger.Log.Warn("Invalid update request", zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if err := h.db.Save(&user).Error; err != nil {
		logger.Log.Error("Failed to update user", zap.Int("id", id), zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	logger.Log.Info("User updated", zap.Int("id", id))
	c.JSON(http.StatusOK, user)
}

// DeleteUser 删除用户
func (h *UserHandler) DeleteUser(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	if err := h.db.Delete(&model.User{}, id).Error; err != nil {
		logger.Log.Error("Failed to delete user", zap.Int("id", id), zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	logger.Log.Info("User deleted", zap.Int("id", id))
	c.JSON(http.StatusNoContent, nil)
}
