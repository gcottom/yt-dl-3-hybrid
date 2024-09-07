package main

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/gcottom/go-zaplog"
	"github.com/gcottom/qgin/qgin"
	"github.com/gcottom/yt-dl-3-hybrid/yd-dl-local-services/yt-dl-local-services-go/config"
	"github.com/gcottom/yt-dl-3-hybrid/yd-dl-local-services/yt-dl-local-services-go/handlers"
	"github.com/gcottom/yt-dl-3-hybrid/yd-dl-local-services/yt-dl-local-services-go/pkg/http_client"
	"github.com/gcottom/yt-dl-3-hybrid/yd-dl-local-services/yt-dl-local-services-go/services/downloader"
	"github.com/gin-contrib/cors"
)

func main() {
	config, err := config.LoadConfigFromFile("")
	if err != nil {
		panic(err)
	}
	if err := RunServer(config); err != nil {
		panic(err)
	}
}

func RunServer(cfg *config.Config) error {
	ctx := zaplog.CreateAndInject(context.Background())
	zaplog.InfoC(ctx, "starting downloader server")

	zaplog.InfoC(ctx, "creating http client")
	httpClient := http_client.NewHTTPClient()

	zaplog.InfoC(ctx, "creating download service")
	downloaderService := downloader.NewDownloaderService(cfg, httpClient)

	go downloaderService.DLQueueProcessor()
	go downloaderService.StatusProcessor()

	zaplog.InfoC(ctx, "creating gin engine")
	ginws := qgin.NewGinEngine(&ctx, &qgin.Config{
		UseContextMW:       true,
		UseLoggingMW:       true,
		UseRequestIDMW:     false,
		InjectRequestIDCTX: false,
		LogRequestID:       false,
		ProdMode:           true,
	})
	ginws.Use(cors.New(cors.Config{
		AllowAllOrigins:  true,
		AllowMethods:     []string{"GET", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Length", "Content-Type", "Accept"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}))

	zaplog.InfoC(ctx, "setting up routes")
	handlers.SetupRoutes(ginws, downloaderService)

	zaplog.InfoC(ctx, fmt.Sprintf("serving on port %d", cfg.LocalPort))
	return http.ListenAndServe(fmt.Sprintf(":%d", cfg.LocalPort), ginws)
}
