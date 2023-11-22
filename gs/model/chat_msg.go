package model

import (
	"go.mongodb.org/mongo-driver/bson/primitive"
)

const (
	ChatMsgTypeText = iota
	ChatMsgTypeIcon
)

type ChatMsg struct {
	ID       primitive.ObjectID `bson:"_id,omitempty"`
	Sequence uint32             `bson:"-"`
	Time     uint32             `bson:"time"`
	Uid      uint32             `bson:"uid"`
	ToUid    uint32             `bson:"to_uid"`
	IsRead   bool               `bson:"is_read"`
	MsgType  uint8              `bson:"msg_type"`
	Text     string             `bson:"text"`
	Icon     uint32             `bson:"icon"`
	IsDelete bool               `bson:"is_delete"`
}
