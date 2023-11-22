package model

import "go.mongodb.org/mongo-driver/bson/primitive"

type Account struct {
	ID                   primitive.ObjectID `bson:"_id,omitempty"`
	AccountId            uint32             `bson:"account_id"`              // 账号id
	Username             string             `bson:"username"`                // 用户名
	Password             string             `bson:"password"`                // 密码
	Token                string             `bson:"token"`                   // 账号token
	TokenCreateTime      uint64             `bson:"token_create_time"`       // 毫秒时间戳
	ComboToken           string             `bson:"combo_token"`             // 游戏服务器token
	ComboTokenCreateTime uint64             `bson:"combo_token_create_time"` // 毫秒时间戳
}
