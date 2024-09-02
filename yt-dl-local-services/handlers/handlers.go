package handlers

import (
	"errors"

	"github.com/gcottom/go-zaplog"
	"github.com/gcottom/yt-dl-3-hybrid/yd-dl-local-service/services/downloader"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type Handler struct {
	DownloaderService downloader.DownloaderService
}

func (h *Handler) StartDownload(ctx *gin.Context) {
	id := ctx.Query("id")
	if id == "" {
		zaplog.WarnC(ctx, "start download request without ID present: ID is required")
		ResponseFailure(ctx, errors.New("start download request without ID present: ID is required"))
		return
	}
	zaplog.InfoC(ctx, "starting download request received", zap.String("id", id))
	if err := h.DownloaderService.InitiateDownload(ctx, id); err != nil {
		zaplog.ErrorC(ctx, "error starting download request", zap.Error(err))
		ResponseFailure(ctx, err)
		return
	}
	zaplog.InfoC(ctx, "start download request queued successfully", zap.String("id", id))
	ResponseSuccess(ctx, StartDownloadResponse{State: "ACK"})
}

func (h *Handler) GetStatus(ctx *gin.Context) {
	id := ctx.Query("id")
	if id == "" {
		zaplog.WarnC(ctx, "get status request without ID present: ID is required")
		ResponseFailure(ctx, errors.New("get status request without ID present: ID is required"))
		return
	}
	zaplog.InfoC(ctx, "getting status request received", zap.String("id", id))
	status, err := h.DownloaderService.GetStatus(ctx, id)
	if err != nil {
		zaplog.ErrorC(ctx, "error getting status request", zap.Error(err))
		ResponseFailure(ctx, err)
		return
	}
	if status == nil {
		zaplog.WarnC(ctx, "status not yet available", zap.String("id", id))
		ResponseSuccess(ctx, StatusUpdate{ID: id, Status: "queued"})
		return
	}
	zaplog.InfoC(ctx, "get status request successful", zap.String("id", id))
	ResponseSuccess(ctx, *status)
}
