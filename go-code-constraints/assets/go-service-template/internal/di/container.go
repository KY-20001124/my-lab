package di

import (
	"github.com/gin-gonic/gin"

	"example.com/go-service-template/internal/api"
	"example.com/go-service-template/internal/api/handler"
	"example.com/go-service-template/internal/manager"
	"example.com/go-service-template/internal/service"
	"example.com/go-service-template/internal/storage/mysql"
)

type App struct {
	router *gin.Engine
}

func New() (*App, func(), error) {
	store := mysql.NewThingStore()
	thingService := service.NewThingService(store)
	thingManager := manager.NewThingManager(thingService)
	thingHandler := handler.NewThingHandler(thingManager)

	router := gin.New()
	api.RegisterRoutes(router, thingHandler)

	return &App{router: router}, func() {}, nil
}

func (a *App) Run(addr string) error {
	return a.router.Run(addr)
}
