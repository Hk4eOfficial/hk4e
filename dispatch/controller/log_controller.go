package controller

import (
	"hk4e/dispatch/model"
	"hk4e/pkg/logger"

	"github.com/gin-gonic/gin"
)

// POST https://log-upload-os.mihoyo.com/sdk/dataUpload HTTP/1.1
func (c *Controller) sdkDataUpload(ctx *gin.Context) {
	ctx.Header("Content-type", "application/json")
	_, _ = ctx.Writer.WriteString("{\"code\":0}")
}

// GET http://log-upload-os.hoyoverse.com/perf/config/verify?device_id=dd664c97f924af747b4576a297c132038be239291651474673768&platform=2&name=DESKTOP-EDUS2DL HTTP/1.1
func (c *Controller) perfConfigVerify(ctx *gin.Context) {
	ctx.Header("Content-type", "application/json")
	_, _ = ctx.Writer.WriteString("{\"code\":0}")
}

// POST http://log-upload-os.hoyoverse.com/perf/dataUpload HTTP/1.1
func (c *Controller) perfDataUpload(ctx *gin.Context) {
	ctx.Header("Content-type", "application/json")
	_, _ = ctx.Writer.WriteString("{\"code\":0}")
}

// POST http://overseauspider.yuanshen.com:8888/log HTTP/1.1
func (c *Controller) log8888(ctx *gin.Context) {
	clientLog := new(model.ClientLog)
	err := ctx.ShouldBindJSON(clientLog)
	if err != nil {
		logger.Error("parse client log error: %v", err)
		return
	}
	_, err = c.db.InsertClientLog(clientLog)
	if err != nil {
		logger.Error("insert client log error: %v", err)
		return
	}
	ctx.Header("Content-type", "application/json")
	_, _ = ctx.Writer.WriteString("{\"code\":0}")
}

// POST http://log-upload-os.hoyoverse.com/crash/dataUpload HTTP/1.1
func (c *Controller) crashDataUpload(ctx *gin.Context) {
	ctx.Header("Content-type", "application/json")
	_, _ = ctx.Writer.WriteString("{\"code\":0}")
}
