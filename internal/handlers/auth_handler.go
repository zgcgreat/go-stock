package handlers

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"

	"go-stock/backend/db"
	"go-stock/backend/models"
	"go-stock/internal/middleware"
)

func Register(c *gin.Context) {
	var registerReq struct {
		Username    string `json:"username" binding:"required,min=3,max=20"`
		Email       string `json:"email" binding:"required,email"`
		Password    string `json:"password" binding:"required,min=6"`
		DisplayName string `json:"display_name,omitempty"`
	}

	if err := c.ShouldBindJSON(&registerReq); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid request data",
			"details": err.Error(),
		})
		return
	}

	var existingUser models.User
	result := db.Dao.Where("username = ? OR email = ?", registerReq.Username, registerReq.Email).First(&existingUser)
	if result.Error == nil {
		c.JSON(http.StatusConflict, gin.H{
			"error":   "User already exists",
			"message": "用户名或邮箱已存在",
		})
		return
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(registerReq.Password), bcrypt.DefaultCost)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to hash password",
			"message": "密码加密失败",
		})
		return
	}

	newUser := &models.User{
		Username:    registerReq.Username,
		Email:       registerReq.Email,
		Password:    string(hashedPassword),
		DisplayName: registerReq.DisplayName,
		IsActive:    true,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	if err := db.Dao.Create(newUser).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to create user",
			"message": "创建用户失败",
		})
		return
	}

	token, err := middleware.GenerateToken(newUser)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to generate token",
			"message": "令牌生成失败",
		})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"message": "User registered successfully",
		"user": gin.H{
			"id":           newUser.ID,
			"username":     newUser.Username,
			"display_name": newUser.DisplayName,
			"email":        newUser.Email,
		},
		"token": token,
	})
}

func Login(c *gin.Context) {
	var loginReq struct {
		Username string `json:"username" binding:"required"`
		Password string `json:"password" binding:"required"`
	}

	if err := c.ShouldBindJSON(&loginReq); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid request data",
			"details": err.Error(),
		})
		return
	}

	var user models.User
	result := db.Dao.Where("username = ? OR email = ?", loginReq.Username, loginReq.Username).First(&user)
	if result.Error != nil {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error":   "User not found",
			"message": "用户名或密码错误",
		})
		return
	}

	if !user.IsActive {
		c.JSON(http.StatusForbidden, gin.H{
			"error":   "Account deactivated",
			"message": "账号已被禁用",
		})
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(loginReq.Password)); err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error":   "Incorrect password",
			"message": "用户名或密码错误",
		})
		return
	}

	token, err := middleware.GenerateToken(&user)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to generate token",
			"message": "令牌生成失败",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Login successful",
		"user": gin.H{
			"id":           user.ID,
			"username":     user.Username,
			"display_name": user.DisplayName,
			"email":        user.Email,
		},
		"token": token,
	})
}

func Logout(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"message": "Logout successful",
	})
}

func GetUserProfile(c *gin.Context) {
	userID, exists := middleware.GetUserIDFromContext(c)
	if !exists {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "User ID not found in context",
			"message": "用户信息异常",
		})
		return
	}

	var user models.User
	result := db.Dao.Select("id, username, display_name, email, created_at, updated_at").Where("id = ?", userID).First(&user)
	if result.Error != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error":   "User not found",
			"message": "用户不存在",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data": gin.H{
			"id":           user.ID,
			"username":     user.Username,
			"display_name": user.DisplayName,
			"email":        user.Email,
			"created_at":   user.CreatedAt,
			"updated_at":   user.UpdatedAt,
		},
	})
}

func UpdateUserProfile(c *gin.Context) {
	userID, exists := middleware.GetUserIDFromContext(c)
	if !exists {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "User ID not found in context",
			"message": "用户信息异常",
		})
		return
	}

	var updateReq struct {
		DisplayName *string `json:"display_name,omitempty"`
		Email       *string `json:"email,omitempty"`
	}

	if err := c.ShouldBindJSON(&updateReq); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid request data",
			"details": err.Error(),
		})
		return
	}

	updateFields := make(map[string]interface{})

	if updateReq.DisplayName != nil {
		updateFields["display_name"] = *updateReq.DisplayName
	}

	if updateReq.Email != nil {
		var existingUser models.User
		result := db.Dao.Where("email = ? AND id != ?", *updateReq.Email, userID).First(&existingUser)
		if result.Error == nil {
			c.JSON(http.StatusConflict, gin.H{
				"error":   "Email already exists",
				"message": "邮箱已被其他用户使用",
			})
			return
		}
		updateFields["email"] = *updateReq.Email
	}

	if len(updateFields) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "No fields to update",
			"message": "没有可更新的字段",
		})
		return
	}

	updateFields["updated_at"] = time.Now()

	result := db.Dao.Model(&models.User{}).Where("id = ?", userID).Updates(updateFields)
	if result.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to update user profile",
			"message": "更新个人信息失败",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "User profile updated successfully",
		"data":    updateFields,
	})
}
