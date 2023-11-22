package config

import (
	"fmt"

	"github.com/BurntSushi/toml"
)

var CONF *Config = nil

// Config 配置
type Config struct {
	HttpPort  int32     `toml:"http_port"`
	Hk4e      Hk4e      `toml:"hk4e"`
	Hk4eRobot Hk4eRobot `toml:"hk4e_robot"`
	Logger    Logger    `toml:"logger"`
	Database  Database  `toml:"database"`
	Redis     Redis     `toml:"redis"`
	MQ        MQ        `toml:"mq"`
}

// Hk4e 原神服务器
type Hk4e struct {
	KcpAddr                 string `toml:"kcp_addr"`                   // kcp地址 该地址只用来注册到节点服务器 填网关的外网地址 网关本地监听为0.0.0.0
	KcpPort                 int32  `toml:"kcp_port"`                   // kcp端口号
	TcpModeEnable           bool   `toml:"tcp_mode_enable"`            // 是否开启tcp模式 需要hook客户端网络库才能支持 共用kcp端口号
	GameDataConfigPath      string `toml:"game_data_config_path"`      // 配置表路径
	ClientProtoProxyEnable  bool   `toml:"client_proto_proxy_enable"`  // 是否开启客户端协议代理功能
	ForwardModeEnable       bool   `toml:"forward_mode_enable"`        // 是否开启网关到机器人的转发功能
	Version                 string `toml:"version"`                    // 支持的客户端协议版本号 三位数字 多个以逗号分隔 如300,310,315,320
	GateTcpMqAddr           string `toml:"gate_tcp_mq_addr"`           // 访问网关tcp直连消息队列的地址 填网关的内网地址
	GateTcpMqPort           int32  `toml:"gate_tcp_mq_port"`           // tcp消息队列端口号
	LoginSdkUrl             string `toml:"login_sdk_url"`              // 网关登录验证token的sdk服务器地址 目前填dispatch的内网地址
	LoginSdkAccountKey      string `toml:"login_sdk_account_key"`      // sdk服务器账号验证的签名密钥
	LoadSceneLuaConfig      bool   `toml:"load_scene_lua_config"`      // 是否加载场景详情LUA配置数据
	DispatchUrl             string `toml:"dispatch_url"`               // 二级dispatch地址 将域名改为dispatch的外网地址
	ForwardRegionUrl        string `toml:"forward_region_url"`         // 转发的一级dispatch地址
	ForwardDispatchUrl      string `toml:"forward_dispatch_url"`       // 转发的二级dispatch地址
	GmAuthKey               string `toml:"gm_auth_key"`                // gm认证密钥
	RegisterAllProtoMessage bool   `toml:"register_all_proto_message"` // 注册全部pb消息
}

// Hk4eRobot 原神机器人
type Hk4eRobot struct {
	RegionListUrl                string `toml:"region_list_url"`                 // 一级dispatch地址
	RegionListParam              string `toml:"region_list_param"`               // 一级dispatch的url参数
	SelectRegionIndex            int32  `toml:"select_region_index"`             // 选择的二级dispatch索引
	CurRegionUrl                 string `toml:"cur_region_url"`                  // 二级dispatch地址 可强制指定 为空则使用一级dispatch获取的地址
	CurRegionParam               string `toml:"cur_region_param"`                // 二级dispatch的url参数
	KeyId                        string `toml:"key_id"`                          // 客户端密钥编号
	LoginSdkUrl                  string `toml:"login_sdk_url"`                   // sdk登录服务器地址
	Account                      string `toml:"account"`                         // 帐号
	Password                     string `toml:"password"`                        // base64编码的rsa公钥加密后的密码
	ClientVersion                string `toml:"client_version"`                  // 客户端版本号
	DosEnable                    bool   `toml:"dos_enable"`                      // 是否开启压力测试
	DosTotalNum                  int32  `toml:"dos_total_num"`                   // 压力测试总并发数量 帐号自动添加后缀编号
	DosBatchNum                  int32  `toml:"dos_batch_num"`                   // 压力测试每批登录并发数量
	DosLoopLogin                 bool   `toml:"dos_loop_login"`                  // 压力测试是否循环登录退出
	ClientMoveEnable             bool   `toml:"client_move_enable"`              // 是否开启客户端模拟移动
	ClientMoveSpeed              int32  `toml:"client_move_speed"`               // 客户端模拟移动速度
	ClientMoveRangeExt           int32  `toml:"client_move_range_ext"`           // 客户端模拟移动区域半径
	ForwardChecksum              string `toml:"forward_checksum"`                // 转发模式强制指定校验和
	ForwardChecksumClientVersion string `toml:"forward_checksum_client_version"` // 转发模式强制指定校验和客户端版本
}

// Logger 日志
type Logger struct {
	Level   string `toml:"level"`
	Mode    string `toml:"mode"`
	Track   bool   `toml:"track"`
	MaxSize int32  `toml:"max_size"`
}

// Database 数据库
type Database struct {
	Url string `toml:"url"`
}

// Redis 缓存
type Redis struct {
	Addr     string `toml:"addr"`
	Password string `toml:"password"`
}

// MQ 消息队列
type MQ struct {
	NatsUrl string `toml:"nats_url"`
}

func InitConfig(filePath string) {
	CONF = new(Config)
	CONF.loadConfigFile(filePath)
}

func GetConfig() *Config {
	return CONF
}

// 加载配置文件
func (c *Config) loadConfigFile(filePath string) {
	_, err := toml.DecodeFile(filePath, &c)
	if err != nil {
		info := fmt.Sprintf("config file load error: %v\n", err)
		panic(info)
	}
}
