package controller

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"hk4e/common/config"
	"hk4e/pkg/logger"

	"github.com/gin-gonic/gin"
)

// gate请求错误响应
func (c *Controller) gateReqErrorRsp(ctx *gin.Context) {
	ctx.JSON(http.StatusOK, gin.H{"retcode": -101, "message": "系统错误", "data": nil})
}

type TokenVerifyReq struct {
	AppID      uint32 `json:"app_id"`
	ChannelID  uint32 `json:"channel_id"`
	OpenID     string `json:"open_id"`
	ComboToken string `json:"combo_token"`
	Sign       string `json:"sign"`
	Region     string `json:"region"`
}

type TokenVerifyRsp struct {
	RetCode int32  `json:"retcode"`
	Message string `json:"message"`
	Data    struct {
		Guest       bool   `json:"guest"`
		AccountType uint32 `json:"account_type"`
		AccountUid  uint32 `json:"account_uid"`
		IpInfo      struct {
			CountryCode string `json:"country_code"`
		} `json:"ip_info"`
	} `json:"data"`
}

func (c *Controller) gateTokenVerify(ctx *gin.Context) {
	tokenVerifyReq := new(TokenVerifyReq)
	err := ctx.ShouldBindJSON(tokenVerifyReq)
	if err != nil {
		c.gateReqErrorRsp(ctx)
		return
	}
	logger.Info("gate token verify, req: %v", tokenVerifyReq)
	signStr := fmt.Sprintf("app_id=%d&channel_id=%d&combo_token=%s&open_id=%s", 1, 1, tokenVerifyReq.ComboToken, tokenVerifyReq.OpenID)
	signHash := hmac.New(sha256.New, []byte(config.GetConfig().Hk4e.LoginSdkAccountKey))
	signHash.Write([]byte(signStr))
	signData := signHash.Sum(nil)
	sign := hex.EncodeToString(signData)
	if tokenVerifyReq.Sign != sign {
		c.gateReqErrorRsp(ctx)
		return
	}
	accountId, err := strconv.Atoi(tokenVerifyReq.OpenID)
	if err != nil {
		c.gateReqErrorRsp(ctx)
		return
	}
	account, err := c.db.QueryAccountByField("account_id", uint64(accountId))
	if err != nil || account == nil {
		c.gateReqErrorRsp(ctx)
		return
	}
	if tokenVerifyReq.ComboToken != account.ComboToken {
		c.gateReqErrorRsp(ctx)
		return
	}
	if time.Now().UnixMilli()-int64(account.ComboTokenCreateTime) > time.Hour.Milliseconds()*24 {
		c.gateReqErrorRsp(ctx)
		return
	}
	ctx.JSON(http.StatusOK, &TokenVerifyRsp{
		RetCode: 0,
		Message: "OK",
		Data: struct {
			Guest       bool   `json:"guest"`
			AccountType uint32 `json:"account_type"`
			AccountUid  uint32 `json:"account_uid"`
			IpInfo      struct {
				CountryCode string `json:"country_code"`
			} `json:"ip_info"`
		}{
			Guest:       false,
			AccountType: 1,
			AccountUid:  uint32(accountId),
			IpInfo: struct {
				CountryCode string `json:"country_code"`
			}{
				CountryCode: "US",
			},
		},
	})
}
