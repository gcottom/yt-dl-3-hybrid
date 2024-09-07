package handlers

import (
	"github.com/gcottom/yt-dl-3-hybrid/yd-dl-local-services/yt-dl-local-services-go/services/downloader"
	"github.com/gin-gonic/gin"
)

func SetupRoutes(router *gin.Engine, downloaderService downloader.DownloaderService) {
	handler := &Handler{DownloaderService: downloaderService}
	router.GET("/download", handler.StartDownload)
	router.GET("/status", handler.GetStatus)
}
