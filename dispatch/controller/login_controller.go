package controller

import (
	"encoding/base64"
	"encoding/json"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"hk4e/dispatch/api"
	"hk4e/dispatch/model"
	"hk4e/pkg/endec"
	"hk4e/pkg/logger"
	"hk4e/pkg/random"

	"github.com/gin-gonic/gin"
)

func (c *Controller) apiLogin(ctx *gin.Context) {
	requestData := new(api.LoginAccountRequestJson)
	err := ctx.ShouldBindJSON(requestData)
	if err != nil {
		logger.Error("parse LoginAccountRequestJson error: %v", err)
		return
	}

	encPwdData, err := base64.StdEncoding.DecodeString(requestData.Password)
	if err != nil {
		logger.Error("decode password enc data error: %v", err)
		return
	}
	pwdPrivKey, err := endec.RsaParsePrivKey(c.pwdRsaKey)
	if err != nil {
		logger.Error("parse rsa key error: %v", err)
		return
	}
	pwdDecData, err := endec.RsaDecrypt(encPwdData, pwdPrivKey)
	useAtAtMode := false
	if err != nil {
		logger.Debug("rsa dec error: %v", err)
		logger.Debug("password rsa dec fail, fallback to @@ mode")
		useAtAtMode = true
	} else {
		logger.Debug("password dec: %v", string(pwdDecData))
		useAtAtMode = false
	}

	responseData := api.NewLoginResult()

	var username string
	var password string
	if useAtAtMode {
		// 账号格式检查 用户名6-20字符 密码8-20字符 用户名和密码公用account字段 第一次出现的@@视为分割标识 username@@password
		if len(requestData.Account) > 20+20+2 {
			responseData.Retcode = -201
			responseData.Message = "用户名或密码长度超限"
			ctx.JSON(http.StatusOK, responseData)
			return
		}
		if !strings.Contains(requestData.Account, "@@") {
			responseData.Retcode = -201
			responseData.Message = "用户名同密码均填写到用户名输入框，填写格式为：用户名@@密码，密码输入框填写任意字符均可"
			ctx.JSON(http.StatusOK, responseData)
			return
		}
		atIndex := strings.Index(requestData.Account, "@@")
		username = requestData.Account[:atIndex]
		password = requestData.Account[atIndex+2:]
	} else {
		username = requestData.Account
		password = string(pwdDecData)
	}

	if len(username) < 6 || len(username) > 20 {
		responseData.Retcode = -201
		responseData.Message = "用户名为6-20位字符"
		ctx.JSON(http.StatusOK, responseData)
		return
	}
	if len(password) < 8 || len(password) > 20 {
		responseData.Retcode = -201
		responseData.Message = "密码为8-20位字符"
		ctx.JSON(http.StatusOK, responseData)
		return
	}
	ok, err := regexp.MatchString("^[a-zA-Z0-9]{6,20}$", username)
	if err != nil || !ok {
		responseData.Retcode = -201
		responseData.Message = "用户名只能包含大小写字母和数字"
		ctx.JSON(http.StatusOK, responseData)
		return
	}
	account, err := c.db.QueryAccountByField("username", username)
	if err != nil {
		logger.Error("query account from db error: %v", err)
		return
	}
	if account == nil {
		// 自动注册
		accountId, err := c.db.GetNextAccountId()
		if err != nil {
			logger.Error("get next account id error: %v", err)
			responseData.Retcode = -201
			responseData.Message = "服务器内部错误:-1"
			ctx.JSON(http.StatusOK, responseData)
			return
		}
		regAccount := &model.Account{
			AccountId:  accountId,
			Username:   username,
			Password:   endec.Md5Str(password),
			Token:      "",
			ComboToken: "",
		}
		_, err = c.db.InsertAccount(regAccount)
		if err != nil {
			logger.Error("insert account error: %v", err)
			responseData.Retcode = -201
			responseData.Message = "服务器内部错误:-2"
			ctx.JSON(http.StatusOK, responseData)
			return
		}
		account = regAccount
	}
	if endec.Md5Str(password) != account.Password {
		responseData.Retcode = -201
		responseData.Message = "用户名或密码错误"
		ctx.JSON(http.StatusOK, responseData)
		return
	}
	// 生成新的token
	account.Token = base64.StdEncoding.EncodeToString(random.GetRandomByte(24))
	_, err = c.db.UpdateAccountFieldByFieldName("account_id", account.AccountId, "token", account.Token)
	if err != nil {
		logger.Error("update account token error: %v", err)
		responseData.Retcode = -201
		responseData.Message = "服务器内部错误:-3"
		ctx.JSON(http.StatusOK, responseData)
		return
	}
	_, err = c.db.UpdateAccountFieldByFieldName("account_id", account.AccountId, "token_create_time", time.Now().UnixMilli())
	if err != nil {
		logger.Error("update account token time error: %v", err)
		responseData.Retcode = -201
		responseData.Message = "服务器内部错误:-4"
		ctx.JSON(http.StatusOK, responseData)
		return
	}
	responseData.Message = "OK"
	responseData.Data.Account.Uid = strconv.FormatInt(int64(account.AccountId), 10)
	responseData.Data.Account.Token = account.Token
	responseData.Data.Account.Email = account.Username
	ctx.JSON(http.StatusOK, responseData)
}

func (c *Controller) apiVerify(ctx *gin.Context) {
	requestData := new(api.LoginTokenRequest)
	err := ctx.ShouldBindJSON(requestData)
	if err != nil {
		logger.Error("parse LoginTokenRequest error: %v", err)
		return
	}
	uid, err := strconv.ParseInt(requestData.Uid, 10, 64)
	if err != nil {
		logger.Error("parse uid error: %v", err)
		return
	}
	account, err := c.db.QueryAccountByField("account_id", uid)
	if err != nil {
		logger.Error("query account from db error: %v", err)
		return
	}
	responseData := api.NewLoginResult()
	if account == nil || account.Token != requestData.Token {
		responseData.Retcode = -111
		responseData.Message = "账号本地缓存信息错误"
		ctx.JSON(http.StatusOK, responseData)
		return
	}
	if uint64(time.Now().UnixMilli())-account.TokenCreateTime > uint64(time.Hour.Milliseconds()*24*7) {
		responseData.Retcode = -111
		responseData.Message = "登录已失效"
		ctx.JSON(http.StatusOK, responseData)
		return
	}
	responseData.Message = "OK"
	responseData.Data.Account.Uid = requestData.Uid
	responseData.Data.Account.Token = requestData.Token
	responseData.Data.Account.Email = account.Username
	ctx.JSON(http.StatusOK, responseData)
}

func (c *Controller) v2Login(ctx *gin.Context) {
	requestData := new(api.ComboTokenReq)
	err := ctx.ShouldBindJSON(requestData)
	if err != nil {
		logger.Error("parse ComboTokenReq error: %v", err)
		return
	}
	data := requestData.Data
	if len(data) == 0 {
		logger.Error("requestData.Data len == 0")
		return
	}
	loginData := new(api.LoginTokenData)
	err = json.Unmarshal([]byte(data), loginData)
	if err != nil {
		logger.Error("Unmarshal LoginTokenData error: %v", err)
		return
	}
	uid, err := strconv.ParseInt(loginData.Uid, 10, 64)
	if err != nil {
		logger.Error("ParseInt uid error: %v", err)
		return
	}
	responseData := api.NewComboTokenRsp()
	account, err := c.db.QueryAccountByField("account_id", uid)
	if account == nil || account.Token != loginData.Token {
		responseData.Retcode = -201
		responseData.Message = "token错误"
		ctx.JSON(http.StatusOK, responseData)
		return
	}
	// 生成新的comboToken
	account.ComboToken = random.GetRandomByteHexStr(20)
	_, err = c.db.UpdateAccountFieldByFieldName("account_id", account.AccountId, "combo_token", account.ComboToken)
	if err != nil {
		logger.Error("update combo token error: %v", err)
		responseData.Retcode = -201
		responseData.Message = "服务器内部错误:-1"
		ctx.JSON(http.StatusOK, responseData)
		return
	}
	_, err = c.db.UpdateAccountFieldByFieldName("account_id", account.AccountId, "combo_token_create_time", time.Now().UnixMilli())
	if err != nil {
		logger.Error("update combo token time error: %v", err)
		responseData.Retcode = -201
		responseData.Message = "服务器内部错误:-2"
		ctx.JSON(http.StatusOK, responseData)
		return
	}
	responseData.Message = "OK"
	responseData.Data.OpenID = loginData.Uid
	responseData.Data.ComboID = "0"
	responseData.Data.ComboToken = account.ComboToken
	ctx.JSON(http.StatusOK, responseData)
}
