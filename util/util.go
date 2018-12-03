package util

import "github.com/gin-gonic/gin"

func NewError(s string, err error) gin.H {
	return gin.H{"message": s, "error": err.Error()}
}
