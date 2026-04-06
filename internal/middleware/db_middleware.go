package middleware

import (
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func DatabaseMiddleware(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Set("db", db)
		c.Next()
	}
}

func GetDBFromContext(c *gin.Context) *gorm.DB {
	db, exists := c.Get("db")
	if !exists {
		return nil
	}

	if database, ok := db.(*gorm.DB); ok {
		return database
	}

	return nil
}
