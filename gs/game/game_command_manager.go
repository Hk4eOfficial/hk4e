package game

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strconv"
	"strings"

	"hk4e/gs/model"
	"hk4e/pkg/logger"
)

// GM命令管理器模块

// CommandPerm 命令权限等级
// 0 为普通玩家 数越大权限越大
type CommandPerm uint8

const (
	CommandPermNormal = CommandPerm(iota) // 普通玩家
	CommandPermGM                         // 管理员
)

// CommandFunc 命令执行函数
type CommandFunc func(content *CommandContent) bool

const (
	PlayerChatGM = iota // 玩家聊天GM
	SystemFuncGM        // 系统函数GM
	DevClientGM         // 开发客户端GM
)

// CommandMessage 命令消息
// 给下层执行命令时提供数据
type CommandMessage struct {
	GMType int // GM类型
	// 玩家聊天GM以及开发客户端GM
	Executor *model.Player // 执行者
	Text     string        // 命令文本
	// 系统函数GM
	FuncName   string            // 函数名
	ParamList  []string          // 函数参数列表
	ResultChan chan *GMCmdResult // 执行结果返回管道
}

type GMCmdResult struct {
	Code int32
	Msg  string
}

// CommandContentStepFunc 命令步骤处理函数
type CommandContentStepFunc func(param any) bool

// CommandContentStepType 命令步骤类型
type CommandContentStepType uint8

const (
	CommandContentStepTypeNone    = CommandContentStepType(iota)
	CommandContentStepTypeDynamic // 动态
	CommandContentStepTypeOption  // 可选
)

// CommandContentStep 命令步骤结构
type CommandContentStep struct {
	StepType     CommandContentStepType // 步骤类型
	ParamTypeStr string                 // 当前步骤参数类型
	StepFunc     CommandContentStepFunc // 步骤处理函数
}

// CommandContent 命令内容
type CommandContent struct {
	Executor     *model.Player      // 执行者
	AssignPlayer *model.Player      // 指定玩家
	Name         string             // 玩家输入的命令名
	ParamList    []string           // 玩家输入的参数列表
	Controller   *CommandController // 命令控制器
	// 执行时数据
	paramIndex uint8                 // 当前执行到的参数索引
	elseFunc   func()                // 参数错误处理函数
	stepList   []*CommandContentStep // 步骤处理函数列表
}

// SendMessage 发送消息
func (c *CommandContent) SendMessage(player *model.Player, msg string, param ...any) {
	GAME.SendPrivateChat(COMMAND_MANAGER.system, player.PlayerId, fmt.Sprintf(msg, param...))
}

// SendColorMessage 发送颜色消息
func (c *CommandContent) SendColorMessage(player *model.Player, color, text string, param ...any) {
	c.SendMessage(player, "<color=%v>%v</color>", color, fmt.Sprintf(text, param...))
}

// SendSuccMessage 发送成功颜色消息
func (c *CommandContent) SendSuccMessage(player *model.Player, text string, param ...any) {
	c.SendColorMessage(player, "#CCFFCC", text, param...)
}

// SendFailMessage 发送失败颜色消息
func (c *CommandContent) SendFailMessage(player *model.Player, text string, param ...any) {
	c.SendColorMessage(player, "#FF9999", text, param...)
}

// getNextParam 获取下一个参数
func (c *CommandContent) getNextParam(typeStr string) (param any, ok bool) {
	// 索引变更
	c.paramIndex++
	// 确保参数长度足够 -1是因为第一次的时候也是获取下一个参数
	if len(c.ParamList) <= int(c.paramIndex)-1 {
		return
	}
	// 获取字符串参数
	paramStr := c.ParamList[c.paramIndex-1]
	// 转换参数类型
	switch typeStr {
	case "int":
		val, err := strconv.ParseInt(paramStr, 10, 64)
		if err != nil {
			return
		}
		return int(val), true
	case "uint":
		val, err := strconv.ParseUint(paramStr, 10, 64)
		if err != nil {
			return
		}
		return uint(val), true
	case "int8":
		val, err := strconv.ParseInt(paramStr, 10, 8)
		if err != nil {
			return
		}
		return int8(val), true
	case "uint8":
		val, err := strconv.ParseUint(paramStr, 10, 8)
		if err != nil {
			return
		}
		return uint8(val), true
	case "int16":
		val, err := strconv.ParseInt(paramStr, 10, 16)
		if err != nil {
			return
		}
		return int16(val), true
	case "uint16":
		val, err := strconv.ParseUint(paramStr, 10, 16)
		if err != nil {
			return
		}
		return uint16(val), true
	case "int32":
		val, err := strconv.ParseInt(paramStr, 10, 32)
		if err != nil {
			return
		}
		return int32(val), true
	case "uint32":
		val, err := strconv.ParseUint(paramStr, 10, 32)
		if err != nil {
			return
		}
		return uint32(val), true
	case "int64":
		val, err := strconv.ParseInt(paramStr, 10, 64)
		if err != nil {
			return
		}
		return val, true
	case "uint64":
		val, err := strconv.ParseUint(paramStr, 10, 64)
		if err != nil {
			return
		}
		return val, true
	case "float32":
		val, err := strconv.ParseFloat(paramStr, 32)
		if err != nil {
			return
		}
		return float32(val), true
	case "float64":
		val, err := strconv.ParseFloat(paramStr, 64)
		if err != nil {
			return
		}
		return val, true
	case "bool":
		val, err := strconv.ParseBool(paramStr)
		if err != nil {
			return
		}
		return val, true
	case "string":
		return paramStr, true
	default:
		return
	}
}

// Dynamic 动态参数执行
func (c *CommandContent) Dynamic(typeStr string, stepFunc CommandContentStepFunc) *CommandContent {
	step := &CommandContentStep{
		StepType:     CommandContentStepTypeDynamic,
		ParamTypeStr: typeStr,
		StepFunc:     stepFunc,
	}
	c.stepList = append(c.stepList, step)
	return c
}

// Option 可选参数执行
func (c *CommandContent) Option(typeStr string, stepFunc CommandContentStepFunc) *CommandContent {
	step := &CommandContentStep{
		StepType:     CommandContentStepTypeOption,
		ParamTypeStr: typeStr,
		StepFunc:     stepFunc,
	}
	c.stepList = append(c.stepList, step)
	return c
}

// Execute 执行命令实际业务并返回结果
func (c *CommandContent) Execute(thenFunc func() bool) bool {
	dynamicStepCount := 0 // 必填参数数量
	dynamicStepIndex := 0 // 必填参数最后的位置
	for i, step := range c.stepList {
		if step.StepType == CommandContentStepTypeDynamic {
			dynamicStepCount++
			dynamicStepIndex = i
		}
	}
	// 执行每个步骤
	for i, step := range c.stepList {
		switch step.StepType {
		case CommandContentStepTypeOption:
			// 可选参数 参数不足则不执行
			if i <= dynamicStepIndex {
				if len(c.ParamList) <= i+dynamicStepCount {
					continue
				}
			} else if len(c.ParamList) <= i {
				continue
			}
		}
		// 获取当前参数
		param, ok := c.getNextParam(step.ParamTypeStr)
		if !ok {
			return false
		}
		// 执行处理函数
		if !step.StepFunc(param) {
			return false
		}
	}
	if thenFunc() {
		return true
	}
	return false
}

// SetElse 设置参数执行错误处理
func (c *CommandContent) SetElse(elseFunc func()) {
	c.elseFunc = elseFunc
}

// CommandManager 命令管理器
type CommandManager struct {
	system                *model.Player                 // GM指令聊天消息机器人
	commandControllerList []*CommandController          // 命令控制器注册列表
	commandControllerMap  map[string]*CommandController // 记录命令控制器
	commandMessageInput   chan *CommandMessage          // 传输要处理的命令消息
	gmCmd                 *GMCmd
	gmCmdRefValue         reflect.Value
}

// NewCommandManager 新建命令管理器
func NewCommandManager() *CommandManager {
	r := new(CommandManager)
	// 初始化
	r.commandControllerList = make([]*CommandController, 0)
	r.commandControllerMap = make(map[string]*CommandController)
	r.commandMessageInput = make(chan *CommandMessage, 1000)
	// 初始化命令控制器
	r.InitController()
	r.gmCmd = new(GMCmd)
	r.gmCmdRefValue = reflect.ValueOf(r.gmCmd)
	return r
}

func (c *CommandManager) GetCommandMessageInput() chan *CommandMessage {
	return c.commandMessageInput
}

// SetSystem 设置GM指令聊天消息机器人
func (c *CommandManager) SetSystem(system *model.Player) {
	c.system = system
}

// RegAllController 注册所有命令控制器
func (c *CommandManager) RegAllController(controllerList ...*CommandController) {
	for _, controller := range controllerList {
		c.RegController(controller)
	}
}

// RegController 注册命令控制器
func (c *CommandManager) RegController(controller *CommandController) {
	// 支持一个命令拥有多个别名
	for _, name := range controller.AliasList {
		// 命令名统一转为小写
		name = strings.ToLower(name)
		// 如果命令已注册则报错 后者覆盖前者
		_, ok := c.commandControllerMap[name]
		if ok {
			// 别名重复注册提示功能
			controller.Func = func(content *CommandContent) bool {
				content.SetElse(func() {
					content.SendFailMessage(content.Executor, "命令别名重复注册，重复的别名：%v。", name)
				})
				return false
			}
			logger.Error("register command repeat, name: %v", name)
		}
		// 记录命令
		c.commandControllerMap[name] = controller
	}
	c.commandControllerList = append(c.commandControllerList, controller)
}

// DelAllController 卸载所有命令控制器
func (c *CommandManager) DelAllController(controllerList ...*CommandController) {
	for _, controller := range controllerList {
		c.DelController(controller)
	}
}

// DelController 卸载命令控制器
func (c *CommandManager) DelController(controller *CommandController) {
	// 支持一个命令拥有多个别名
	for _, name := range controller.AliasList {
		delete(c.commandControllerMap, name)
	}
	// 卸载列表上的控制器
	for i, commandController := range c.commandControllerList {
		if commandController == controller {
			c.commandControllerList = append(c.commandControllerList[:i], c.commandControllerList[i+1:]...)
		}
	}
}

// PlayerInputCommand 玩家输入要处理的命令
func (c *CommandManager) PlayerInputCommand(player *model.Player, targetUid uint32, text string) {
	// 机器人不会读命令所以写到了 PrivateChatReq

	// 确保私聊的目标是处理命令的机器人
	if targetUid != c.system.PlayerId {
		return
	}

	world := WORLD_MANAGER.GetWorldById(player.WorldId)
	if world != nil && WORLD_MANAGER.IsAiWorld(world) {
		return
	}

	// 输入的命令将在主协程中处理
	c.commandMessageInput <- &CommandMessage{
		GMType:   PlayerChatGM,
		Executor: player,
		Text:     text,
	}
}

// CallGMCmd 调用GM命令
func (c *CommandManager) CallGMCmd(funcName string, paramList []string) (bool, string) {
	fn := c.gmCmdRefValue.MethodByName(funcName)
	if !fn.IsValid() {
		logger.Error("gm func not valid, func: %v", funcName)
		return false, ""
	}
	if fn.Type().NumIn() != len(paramList) {
		logger.Error("gm func param num not match, func: %v, need: %v, give: %v", funcName, fn.Type().NumIn(), len(paramList))
		return false, ""
	}
	in := make([]reflect.Value, fn.Type().NumIn())
	for i := 0; i < fn.Type().NumIn(); i++ {
		kind := fn.Type().In(i).Kind()
		param := paramList[i]
		var value reflect.Value
		switch kind {
		case reflect.Int:
			val, err := strconv.ParseInt(param, 10, 64)
			if err != nil {
				return false, ""
			}
			value = reflect.ValueOf(int(val))
		case reflect.Uint:
			val, err := strconv.ParseUint(param, 10, 64)
			if err != nil {
				return false, ""
			}
			value = reflect.ValueOf(uint(val))
		case reflect.Int8:
			val, err := strconv.ParseInt(param, 10, 8)
			if err != nil {
				return false, ""
			}
			value = reflect.ValueOf(int8(val))
		case reflect.Uint8:
			val, err := strconv.ParseUint(param, 10, 8)
			if err != nil {
				return false, ""
			}
			value = reflect.ValueOf(uint8(val))
		case reflect.Int16:
			val, err := strconv.ParseInt(param, 10, 16)
			if err != nil {
				return false, ""
			}
			value = reflect.ValueOf(int16(val))
		case reflect.Uint16:
			val, err := strconv.ParseUint(param, 10, 16)
			if err != nil {
				return false, ""
			}
			value = reflect.ValueOf(uint16(val))
		case reflect.Int32:
			val, err := strconv.ParseInt(param, 10, 32)
			if err != nil {
				return false, ""
			}
			value = reflect.ValueOf(int32(val))
		case reflect.Uint32:
			val, err := strconv.ParseUint(param, 10, 32)
			if err != nil {
				return false, ""
			}
			value = reflect.ValueOf(uint32(val))
		case reflect.Int64:
			val, err := strconv.ParseInt(param, 10, 64)
			if err != nil {
				return false, ""
			}
			value = reflect.ValueOf(val)
		case reflect.Uint64:
			val, err := strconv.ParseUint(param, 10, 64)
			if err != nil {
				return false, ""
			}
			value = reflect.ValueOf(val)
		case reflect.Float32:
			val, err := strconv.ParseFloat(param, 32)
			if err != nil {
				return false, ""
			}
			value = reflect.ValueOf(float32(val))
		case reflect.Float64:
			val, err := strconv.ParseFloat(param, 64)
			if err != nil {
				return false, ""
			}
			value = reflect.ValueOf(val)
		case reflect.Bool:
			val, err := strconv.ParseBool(param)
			if err != nil {
				return false, ""
			}
			value = reflect.ValueOf(val)
		case reflect.String:
			value = reflect.ValueOf(param)
		default:
			return false, ""
		}
		in[i] = value
	}
	out := fn.Call(in)
	ret := make([]any, 0)
	for _, v := range out {
		ret = append(ret, v.Interface())
	}
	data, _ := json.Marshal(ret)
	return true, string(data)
}

// HandleCommand 处理命令
// 主协程接收到命令消息后执行
func (c *CommandManager) HandleCommand(command *CommandMessage) {
	switch command.GMType {
	case PlayerChatGM, DevClientGM:
		logger.Info("run gm cmd, text: %v, uid: %v", command.Text, command.Executor.PlayerId)
		// 执行命令
		c.ExecCommand(command)
	case SystemFuncGM:
		logger.Info("run gm func, funcName: %v, paramList: %v", command.FuncName, command.ParamList)
		// 反射调用game_command_gm.go中的函数并反射解析传入参数类型
		ok, ret := c.CallGMCmd(command.FuncName, command.ParamList)
		if command.ResultChan != nil {
			var gmCmdResult *GMCmdResult = nil
			if ok {
				gmCmdResult = &GMCmdResult{Code: 0, Msg: ret}
			} else {
				gmCmdResult = &GMCmdResult{Code: -1, Msg: ""}
			}
			command.ResultChan <- gmCmdResult
			close(command.ResultChan)
		}
	}
}

// ExecCommand 执行命令
func (c *CommandManager) ExecCommand(cmd *CommandMessage) {
	// 命令内容
	content := new(CommandContent)
	content.Executor = cmd.Executor
	// 默认指定玩家为执行者
	content.AssignPlayer = cmd.Executor

	// 分割出命令的每个参数
	cmdSplit := strings.Split(cmd.Text, " ")
	// 分割出来啥也没有可能是个空的字符串
	// 此时将会返回的命令名和命令参数都为空
	if len(cmdSplit) == 0 {
		content.SendFailMessage(content.Executor, "命令错误：命令名为空。")
		return
	}
	// 有些命令没有参数 也要适配
	var paramList []string
	if len(cmdSplit) >= 2 {
		paramList = cmdSplit[1:]
	}
	// 不区分命令名的大小写 统一转为小写
	content.Name = strings.ToLower(cmdSplit[0]) // 首个参数必是命令名
	content.ParamList = paramList               // 命令名后当然是命令的参数喽

	// 判断命令是否注册
	controller, ok := c.commandControllerMap[content.Name]
	if !ok {
		// 玩家可能会执行一些没有的命令仅做调试输出
		content.SendFailMessage(content.Executor, "命令 %v 不存在，输入 help 查看帮助。", cmd.Text)
		return
	}
	// 设置控制器
	content.Controller = controller
	// 判断玩家的权限是否符合要求
	player := content.Executor
	if ok && player.CmdPerm < uint8(controller.Perm) {
		content.SendFailMessage(content.Executor, "权限不足，该命令需要%v级权限。\n你目前的权限等级：%v", controller.Perm, player.CmdPerm)
		return
	}
	// 命令指定uid
	if player.CommandAssignUid != 0 {
		// 判断指定玩家是否在线
		target := USER_MANAGER.GetOnlineUser(player.CommandAssignUid)
		// 目标玩家属于非本地玩家
		if target == nil {
			content.SendFailMessage(content.Executor, "命令执行失败，指定玩家离线或不在当前服务器。")
			return
		}
		content.AssignPlayer = target
	}
	// 执行命令
	if controller.Func(content) {
		// 命令执行过程中没有问题就跳出
		return
	}
	// 命令参数错误处理
	if content.elseFunc == nil {
		// 默认的错误处理
		usage := "命令用法：\n"
		for i, s := range controller.UsageList {
			s = strings.ReplaceAll(s, "{alias}", content.Name)
			usage += fmt.Sprintf("%v. %v", i+1, s)
			// 换行
			if i != len(controller.UsageList)-1 {
				usage += "\n"
			}
		}
		content.SendFailMessage(content.Executor, "参数或格式错误，正确用法：\n\n<color=white>%v</color>", usage)
	} else {
		// 自定义的错误处理
		content.elseFunc()
	}
}
