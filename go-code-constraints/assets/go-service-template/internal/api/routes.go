package api

import (
	"github.com/gin-gonic/gin"

	"example.com/go-service-template/internal/api/handler"
)

func RegisterRoutes(r *gin.Engine, thingHandler *handler.ThingHandler) {
	v1 := r.Group("/api/v1")
	v1.GET("/things/:id", thingHandler.Get)
}
