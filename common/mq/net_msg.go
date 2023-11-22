package mq

import pb "google.golang.org/protobuf/proto"

const (
	MsgTypeGame     = iota // 来自客户端的游戏消息
	MsgTypeConnCtrl        // GATE客户端连接信息消息
	MsgTypeServer          // 服务器之间转发的消息
)

type NetMsg struct {
	MsgType           uint8
	EventId           uint16
	ServerType        string `msgpack:"-"`
	AppId             string `msgpack:"-"`
	Topic             string `msgpack:"-"`
	GameMsg           *GameMsg
	ConnCtrlMsg       *ConnCtrlMsg
	ServerMsg         *ServerMsg
	OriginServerType  string
	OriginServerAppId string
}

const (
	NormalMsg = iota // 正常的游戏消息
)

type GameMsg struct {
	UserId             uint32
	CmdId              uint16
	ClientSeq          uint32
	PayloadMessage     pb.Message `msgpack:"-"`
	PayloadMessageData []byte
}

const (
	ClientRttNotify   = iota // 客户端网络时延上报
	KickPlayerNotify         // 通知GATE剔除玩家
	UserOfflineNotify        // 玩家离线通知GS
)

type ConnCtrlMsg struct {
	UserId     uint32
	ClientRtt  uint32
	KickUserId uint32
	KickReason uint32
}

const (
	ServerAppidBindNotify              = iota // 玩家连接绑定的各个服务器appid通知
	ServerUserOnlineStateChangeNotify         // 广播玩家上线和离线状态以及所在GS的appid
	ServerUserGsChangeNotify                  // 跨服玩家迁移通知
	ServerPlayerMpReq                         // 跨服多人世界相关请求
	ServerPlayerMpRsp                         // 跨服多人世界相关响应
	ServerChatMsgNotify                       // 跨服玩家聊天消息通知
	ServerAddFriendNotify                     // 跨服添加好友通知
	ServerForwardModeClientConnNotify         // 转发模式客户端连接通知
	ServerForwardModeClientCloseNotify        // 转发模式客户端断开连接通知
	ServerForwardModeServerCloseNotify        // 转发模式服务器断开连接通知
	ServerForwardDispatchInfoNotify           // 转发模式区服信息通知
	ServerStopNotify                          // 停服通知
	ServerDispatchCancelNotify                // 服务器取消调度通知
	ServerGmCmdNotify                         // 服务器GM指令执行通知
)

type ServerMsg struct {
	MultiServerAppId    string
	UserId              uint32
	IsOnline            bool
	GameServerAppId     string
	JoinHostUserId      uint32
	PlayerMpInfo        *PlayerMpInfo
	ChatMsgInfo         *ChatMsgInfo
	AddFriendInfo       *AddFriendInfo
	ForwardDispatchInfo *ForwardDispatchInfo
	AppVersion          string
	GmCmdFuncName       string
	GmCmdParamList      []string
}

type OriginInfo struct {
	CmdName string
	UserId  uint32
}

type PlayerBaseInfo struct {
	UserId         uint32
	Nickname       string
	PlayerLevel    uint32
	MpSettingType  uint8
	NameCardId     uint32
	Signature      string
	HeadImageId    uint32
	WorldPlayerNum uint32
	WorldLevel     uint32
}

type PlayerMpInfo struct {
	OriginInfo            *OriginInfo
	HostUserId            uint32
	ApplyUserId           uint32
	ApplyPlayerOnlineInfo *PlayerBaseInfo
	ApplyOk               bool
	Agreed                bool
	Reason                int32
	HostNickname          string
}

type ChatMsgInfo struct {
	Time     uint32
	ToUid    uint32
	Uid      uint32
	IsRead   bool
	MsgType  uint8
	Text     string
	Icon     uint32
	IsDelete bool
}

type AddFriendInfo struct {
	OriginInfo            *OriginInfo
	TargetUserId          uint32
	ApplyPlayerOnlineInfo *PlayerBaseInfo
}

type ForwardDispatchInfo struct {
	GateIp      string
	GatePort    uint32
	DispatchKey []byte
}
