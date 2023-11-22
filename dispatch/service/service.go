package service

import (
	"hk4e/dispatch/dao"
)

type Service struct {
	dao *dao.Dao
}

// UserPasswordChange 用户密码改变
func (s *Service) UserPasswordChange(accountId uint32) bool {
	// http登录态失效
	_, err := s.dao.UpdateAccountFieldByFieldName("account_id", accountId, "token_create_time", 0)
	if err != nil {
		return false
	}
	// TODO 游戏内登录态失效
	return true
}
