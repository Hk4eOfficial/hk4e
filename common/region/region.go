package region

import (
	"os"
	"regexp"
	"strconv"

	"hk4e/pkg/logger"
)

func LoadRegionRsaKey() (signRsaKey []byte, encRsaKeyMap map[string][]byte, pwdRsaKey []byte) {
	var err error = nil
	encRsaKeyMap = make(map[string][]byte)
	signRsaKey, err = os.ReadFile("key/region_sign_key.pem")
	if err != nil {
		logger.Error("open region_sign_key.pem error: %v", err)
		return nil, nil, nil
	}
	encKeyIdList := []string{"1", "2", "3", "4", "5"}
	for _, v := range encKeyIdList {
		encRsaKeyMap[v], err = os.ReadFile("key/region_enc_key_" + v + ".pem")
		if err != nil {
			logger.Error("open region_enc_key_"+v+".pem error: %v", err)
			return nil, nil, nil
		}
	}
	pwdRsaKey, err = os.ReadFile("key/account_password_key.pem")
	if err != nil {
		logger.Error("open account_password_key.pem error: %v", err)
		return nil, nil, nil
	}
	return signRsaKey, encRsaKeyMap, pwdRsaKey
}

// GetClientVersionByName 从客户端版本字符串中提取版本号
func GetClientVersionByName(versionName string) (int, string) {
	reg, err := regexp.Compile("[0-9]+")
	if err != nil {
		logger.Error("compile regexp error: %v", err)
		return 0, ""
	}
	versionSlice := reg.FindAllString(versionName, -1)
	version := 0
	for index, value := range versionSlice {
		v, err := strconv.Atoi(value)
		if err != nil {
			logger.Error("parse client version error: %v", err)
			return 0, ""
		}
		if v >= 10 {
			// 测试版本
			if index != 2 {
				logger.Error("invalid client version")
				return 0, ""
			}
			v /= 10
		}
		for i := 0; i < 2-index; i++ {
			v *= 10
		}
		version += v
	}
	return version, strconv.Itoa(version)
}
