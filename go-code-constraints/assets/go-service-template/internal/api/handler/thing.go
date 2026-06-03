package handler

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"example.com/go-service-template/internal/manager"
	"example.com/go-service-template/internal/service"
)

type ThingHandler struct {
	things manager.ThingManager
}

func NewThingHandler(things manager.ThingManager) *ThingHandler {
	return &ThingHandler{things: things}
}

func (h *ThingHandler) Get(c *gin.Context) {
	id := c.Param("id")
	out, err := h.things.Get(c.Request.Context(), id)
	if err != nil {
		if errors.Is(err, service.ErrThingNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "thing not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal error"})
		return
	}
	c.JSON(http.StatusOK, out)
}
