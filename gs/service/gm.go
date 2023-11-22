package service

import (
	"context"
	"time"

	"hk4e/gs/api"
	"hk4e/gs/game"
)

var _ api.GMNATSRPCServer = (*GMService)(nil)

type GMService struct {
	g *game.Game
}

func (s *GMService) Cmd(ctx context.Context, req *api.CmdRequest) (*api.CmdReply, error) {
	commandTextInput := game.COMMAND_MANAGER.GetCommandMessageInput()
	resultChan := make(chan *game.GMCmdResult)
	commandTextInput <- &game.CommandMessage{
		GMType:     game.SystemFuncGM,
		FuncName:   req.FuncName,
		ParamList:  req.ParamList,
		ResultChan: resultChan,
	}
	timer := time.NewTimer(time.Second * 10)
	var cmdReply *api.CmdReply = nil
	select {
	case <-timer.C:
		cmdReply = &api.CmdReply{Code: -1, Message: "执行结果等待超时"}
	case result := <-resultChan:
		cmdReply = &api.CmdReply{Code: result.Code, Message: result.Msg}
	}
	timer.Stop()
	return cmdReply, nil
}
