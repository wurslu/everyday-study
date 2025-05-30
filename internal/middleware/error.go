package middleware

import (
	"everyday-study-backend/internal/models"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
)

func ErrorHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if err := recover(); err != nil {
				log.Printf("Panic recovered: %v", err)
				c.JSON(http.StatusInternalServerError, models.APIResponse{
					Success:   false,
					Message:   "服务器内部错误",
					ErrorCode: "SERVER_ERROR",
				})
				c.Abort()
			}
		}()

		c.Next()

		if len(c.Errors) > 0 {
			err := c.Errors.Last()
			
			switch err.Type {
			case gin.ErrorTypeBind:
				c.JSON(http.StatusBadRequest, models.APIResponse{
					Success:   false,
					Message:   "请求参数错误",
					ErrorCode: "VALIDATION_ERROR",
					Errors:    []string{err.Error()},
				})
			default:
				c.JSON(http.StatusInternalServerError, models.APIResponse{
					Success:   false,
					Message:   "服务器内部错误",
					ErrorCode: "SERVER_ERROR",
				})
			}
		}
	}
}