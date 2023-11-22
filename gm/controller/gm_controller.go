package controller

import (
	"net/http"

	"hk4e/common/mq"
	"hk4e/common/rpc"
	"hk4e/gs/api"
	"hk4e/pkg/logger"

	"github.com/gin-gonic/gin"
)

type GmCmdReq struct {
	FuncName  string   `json:"func_name"`
	ParamList []string `json:"param_list"`
	GsId      uint32   `json:"gs_id"`
	GsAppid   string   `json:"gs_appid"`
}

type GmCmdRsp struct {
	ResultCode int32  `json:"result_code"`
	ResultMsg  string `json:"result_msg"`
	Desc       string `json:"desc"`
}

func (c *Controller) gmCmd(ctx *gin.Context) {
	gmCmdReq := new(GmCmdReq)
	err := ctx.ShouldBindJSON(gmCmdReq)
	if err != nil {
		logger.Error("parse json error: %v", err)
		ctx.JSON(http.StatusOK, &CommonRsp{Code: -1, Msg: "参数解析错误", Data: err})
		return
	}
	logger.Info("GmCmdReq: %v", gmCmdReq)
	if gmCmdReq.GsId != 0 {
		// 指定GSID执行
		c.gmClientMapLock.RLock()
		gmClient, exist := c.gmClientMap[gmCmdReq.GsId]
		c.gmClientMapLock.RUnlock()
		if !exist {
			var err error = nil
			gmClient, err = rpc.NewGMClient(gmCmdReq.GsId)
			if err != nil {
				logger.Error("new gm client error: %v", err)
				ctx.JSON(http.StatusOK, &CommonRsp{Code: -1, Msg: "服务器内部错误", Data: err})
				return
			}
			c.gmClientMapLock.Lock()
			c.gmClientMap[gmCmdReq.GsId] = gmClient
			c.gmClientMapLock.Unlock()
		}
		rsp, err := gmClient.Cmd(ctx.Request.Context(), &api.CmdRequest{
			FuncName:  gmCmdReq.FuncName,
			ParamList: gmCmdReq.ParamList,
		})
		if err != nil {
			ctx.JSON(http.StatusOK, &CommonRsp{Code: -1, Msg: "服务器内部错误", Data: err})
			return
		}
		ctx.JSON(http.StatusOK, &CommonRsp{Code: 0, Msg: "", Data: &GmCmdRsp{ResultCode: rsp.Code, ResultMsg: rsp.Message, Desc: "指定GSID执行"}})
	} else if gmCmdReq.GsAppid != "" {
		// 指定GSAPPID执行
		c.messageQueue.SendToGs(gmCmdReq.GsAppid, &mq.NetMsg{
			MsgType: mq.MsgTypeServer,
			EventId: mq.ServerGmCmdNotify,
			ServerMsg: &mq.ServerMsg{
				GmCmdFuncName:  gmCmdReq.FuncName,
				GmCmdParamList: gmCmdReq.ParamList,
			},
		})
		ctx.JSON(http.StatusOK, &CommonRsp{Code: 0, Msg: "", Data: &GmCmdRsp{ResultCode: 0, ResultMsg: "", Desc: "指定GSAPPID执行"}})
	} else {
		// 全服GS执行
		c.messageQueue.SendToAll(&mq.NetMsg{
			MsgType: mq.MsgTypeServer,
			EventId: mq.ServerGmCmdNotify,
			ServerMsg: &mq.ServerMsg{
				GmCmdFuncName:  gmCmdReq.FuncName,
				GmCmdParamList: gmCmdReq.ParamList,
			},
		})
		ctx.JSON(http.StatusOK, &CommonRsp{Code: 0, Msg: "", Data: &GmCmdRsp{ResultCode: 0, ResultMsg: "", Desc: "全服GS执行"}})
	}
}
