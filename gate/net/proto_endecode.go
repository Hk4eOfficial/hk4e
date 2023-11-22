package net

import (
	"reflect"
	"strings"

	"hk4e/common/config"
	"hk4e/gate/client_proto"
	"hk4e/pkg/logger"
	"hk4e/pkg/object"
	"hk4e/protocol/cmd"
	"hk4e/protocol/proto"

	pb "google.golang.org/protobuf/proto"
)

// pb协议编解码

type ProtoMsg struct {
	SessionId      uint32
	CmdId          uint16
	HeadMessage    *proto.PacketHead
	PayloadMessage pb.Message
}

type ProtoMessage struct {
	cmdId   uint16
	message pb.Message
}

func ProtoDecode(kcpMsg *KcpMsg,
	serverCmdProtoMap *cmd.CmdProtoMap, clientCmdProtoMap *client_proto.ClientCmdProtoMap) (protoMsgList []*ProtoMsg) {
	protoMsgList = make([]*ProtoMsg, 0)
	if config.GetConfig().Hk4e.ClientProtoProxyEnable {
		clientCmdId := kcpMsg.CmdId
		clientProtoData := kcpMsg.ProtoData
		cmdName := clientCmdProtoMap.GetClientCmdNameByCmdId(clientCmdId)
		if cmdName == "" {
			logger.Error("get cmdName is nil, clientCmdId: %v", clientCmdId)
			return protoMsgList
		}
		clientProtoObj := GetClientProtoObjByName(cmdName, clientCmdProtoMap)
		if clientProtoObj == nil {
			logger.Error("get client proto obj is nil, cmdName: %v", cmdName)
			return protoMsgList
		}
		err := pb.Unmarshal(clientProtoData, clientProtoObj)
		if err != nil {
			logger.Error("unmarshal client proto error: %v", err)
			return protoMsgList
		}
		serverCmdId := serverCmdProtoMap.GetCmdIdByCmdName(cmdName)
		if serverCmdId == 0 {
			logger.Error("get server cmdId is nil, cmdName: %v", cmdName)
			return protoMsgList
		}
		serverProtoObj := serverCmdProtoMap.GetProtoObjByCmdId(serverCmdId)
		if serverProtoObj == nil {
			logger.Error("get server proto obj is nil, serverCmdId: %v", serverCmdId)
			return protoMsgList
		}
		err = object.CopyProtoBufSameField(serverProtoObj, clientProtoObj)
		if err != nil {
			logger.Error("copy proto obj error: %v", err)
			return protoMsgList
		}
		ConvClientPbDataToServer(serverProtoObj, clientCmdProtoMap)
		serverProtoData, err := pb.Marshal(serverProtoObj)
		if err != nil {
			logger.Error("marshal server proto error: %v", err)
			return protoMsgList
		}
		kcpMsg.CmdId = serverCmdId
		kcpMsg.ProtoData = serverProtoData
	}
	protoMsg := new(ProtoMsg)
	protoMsg.SessionId = kcpMsg.SessionId
	protoMsg.CmdId = kcpMsg.CmdId
	// head msg
	if kcpMsg.HeadData != nil && len(kcpMsg.HeadData) != 0 {
		headMsg := new(proto.PacketHead)
		err := pb.Unmarshal(kcpMsg.HeadData, headMsg)
		if err != nil {
			logger.Error("unmarshal head data err: %v", err)
			return protoMsgList
		}
		protoMsg.HeadMessage = headMsg
	} else {
		protoMsg.HeadMessage = nil
	}
	// payload msg
	protoMessageList := make([]*ProtoMessage, 0)
	ProtoDecodePayloadLoop(kcpMsg.CmdId, kcpMsg.ProtoData, &protoMessageList, serverCmdProtoMap, clientCmdProtoMap)
	if len(protoMessageList) == 0 {
		logger.Error("decode proto object is nil")
		return protoMsgList
	}
	if kcpMsg.CmdId == cmd.UnionCmdNotify && !config.GetConfig().Hk4e.ForwardModeEnable {
		for _, protoMessage := range protoMessageList {
			msg := new(ProtoMsg)
			msg.SessionId = kcpMsg.SessionId
			msg.CmdId = protoMessage.cmdId
			msg.HeadMessage = protoMsg.HeadMessage
			msg.PayloadMessage = protoMessage.message
			protoMsgList = append(protoMsgList, msg)
		}
		// for _, msg := range protoMsgList {
		// 	cmdName := "???"
		// 	if msg.PayloadMessage != nil {
		// 		cmdName = string(msg.PayloadMessage.ProtoReflect().Descriptor().FullName())
		// 	}
		// 	logger.Debug("[RECV UNION CMD] cmdId: %v, cmdName: %v, sessionId: %v, headMsg: %v",
		// 		msg.CmdId, cmdName, msg.SessionId, msg.HeadMessage)
		// }
	} else {
		protoMsg.PayloadMessage = protoMessageList[0].message
		protoMsgList = append(protoMsgList, protoMsg)
		cmdName := ""
		if protoMsg.PayloadMessage != nil {
			cmdName = string(protoMsg.PayloadMessage.ProtoReflect().Descriptor().FullName())
		}
		logger.Debug("[RECV] cmdId: %v, cmdName: %v, sessionId: %v, headMsg: %v",
			protoMsg.CmdId, cmdName, protoMsg.SessionId, protoMsg.HeadMessage)
	}
	return protoMsgList
}

func ProtoDecodePayloadLoop(cmdId uint16, protoData []byte, protoMessageList *[]*ProtoMessage,
	serverCmdProtoMap *cmd.CmdProtoMap, clientCmdProtoMap *client_proto.ClientCmdProtoMap) {
	protoObj := DecodePayloadToProto(cmdId, protoData, serverCmdProtoMap)
	if protoObj == nil {
		logger.Error("decode proto object is nil")
		return
	}
	if cmdId == cmd.UnionCmdNotify && !config.GetConfig().Hk4e.ForwardModeEnable {
		// 处理聚合消息
		unionCmdNotify, ok := protoObj.(*proto.UnionCmdNotify)
		if !ok {
			logger.Error("parse union cmd error")
			return
		}
		for _, unionCmd := range unionCmdNotify.GetCmdList() {
			if config.GetConfig().Hk4e.ClientProtoProxyEnable {
				clientCmdId := uint16(unionCmd.MessageId)
				clientProtoData := unionCmd.Body
				cmdName := clientCmdProtoMap.GetClientCmdNameByCmdId(clientCmdId)
				if cmdName == "" {
					logger.Error("get cmdName is nil, clientCmdId: %v", clientCmdId)
					continue
				}
				clientProtoObj := GetClientProtoObjByName(cmdName, clientCmdProtoMap)
				if clientProtoObj == nil {
					logger.Error("get client proto obj is nil, cmdName: %v", cmdName)
					continue
				}
				err := pb.Unmarshal(clientProtoData, clientProtoObj)
				if err != nil {
					logger.Error("unmarshal client proto error: %v", err)
					continue
				}
				serverCmdId := serverCmdProtoMap.GetCmdIdByCmdName(cmdName)
				if serverCmdId == 0 {
					logger.Error("get server cmdId is nil, cmdName: %v", cmdName)
					continue
				}
				serverProtoObj := serverCmdProtoMap.GetProtoObjByCmdId(serverCmdId)
				if serverProtoObj == nil {
					logger.Error("get server proto obj is nil, serverCmdId: %v", serverCmdId)
					continue
				}
				err = object.CopyProtoBufSameField(serverProtoObj, clientProtoObj)
				if err != nil {
					logger.Error("copy proto obj error: %v", err)
					continue
				}
				ConvClientPbDataToServer(serverProtoObj, clientCmdProtoMap)
				serverProtoData, err := pb.Marshal(serverProtoObj)
				if err != nil {
					logger.Error("marshal server proto error: %v", err)
					continue
				}
				unionCmd.MessageId = uint32(serverCmdId)
				unionCmd.Body = serverProtoData
			}
			ProtoDecodePayloadLoop(uint16(unionCmd.MessageId), unionCmd.Body, protoMessageList,
				serverCmdProtoMap, clientCmdProtoMap)
		}
	}
	*protoMessageList = append(*protoMessageList, &ProtoMessage{
		cmdId:   cmdId,
		message: protoObj,
	})
}

func ProtoEncode(protoMsg *ProtoMsg,
	serverCmdProtoMap *cmd.CmdProtoMap, clientCmdProtoMap *client_proto.ClientCmdProtoMap) (kcpMsg *KcpMsg) {
	cmdName := ""
	if protoMsg.PayloadMessage != nil {
		cmdName = string(protoMsg.PayloadMessage.ProtoReflect().Descriptor().FullName())
	}
	logger.Debug("[SEND] cmdId: %v, cmdName: %v, sessionId: %v, headMsg: %v",
		protoMsg.CmdId, cmdName, protoMsg.SessionId, protoMsg.HeadMessage)
	kcpMsg = new(KcpMsg)
	kcpMsg.SessionId = protoMsg.SessionId
	kcpMsg.CmdId = protoMsg.CmdId
	// head msg
	if protoMsg.HeadMessage != nil {
		headData, err := pb.Marshal(protoMsg.HeadMessage)
		if err != nil {
			logger.Error("marshal head data err: %v", err)
			return nil
		}
		kcpMsg.HeadData = headData
	} else {
		kcpMsg.HeadData = nil
	}
	if protoMsg.CmdId == cmd.UnionCmdNotify && config.GetConfig().Hk4e.ForwardModeEnable && config.GetConfig().Hk4e.ClientProtoProxyEnable {
		// 处理聚合消息
		unionCmdNotify, ok := protoMsg.PayloadMessage.(*proto.UnionCmdNotify)
		if !ok {
			logger.Error("parse union cmd error")
			return
		}
		for _, unionCmd := range unionCmdNotify.GetCmdList() {
			serverCmdId := uint16(unionCmd.MessageId)
			serverProtoData := unionCmd.Body
			serverProtoObj := serverCmdProtoMap.GetProtoObjByCmdId(serverCmdId)
			if serverProtoObj == nil {
				logger.Error("get server proto obj is nil, serverCmdId: %v", serverCmdId)
				continue
			}
			err := pb.Unmarshal(serverProtoData, serverProtoObj)
			if err != nil {
				logger.Error("unmarshal server proto error: %v", err)
				continue
			}
			ConvServerPbDataToClient(serverProtoObj, clientCmdProtoMap)
			cmdName = serverCmdProtoMap.GetCmdNameByCmdId(serverCmdId)
			if cmdName == "" {
				logger.Error("get cmdName is nil, serverCmdId: %v", serverCmdId)
				continue
			}
			clientProtoObj := GetClientProtoObjByName(cmdName, clientCmdProtoMap)
			if clientProtoObj == nil {
				logger.Error("get client proto obj is nil, cmdName: %v", cmdName)
				continue
			}
			err = object.CopyProtoBufSameField(clientProtoObj, serverProtoObj)
			if err != nil {
				logger.Error("copy proto obj error: %v", err)
				continue
			}
			clientProtoData, err := pb.Marshal(clientProtoObj)
			if err != nil {
				logger.Error("marshal server proto error: %v", err)
				continue
			}
			clientCmdId := clientCmdProtoMap.GetClientCmdIdByCmdName(cmdName)
			if clientCmdId == 0 {
				logger.Error("get client cmdId is nil, cmdName: %v", cmdName)
				continue
			}
			unionCmd.MessageId = uint32(clientCmdId)
			unionCmd.Body = clientProtoData
		}
	}
	// payload msg
	if protoMsg.PayloadMessage != nil {
		protoData := EncodeProtoToPayload(protoMsg.PayloadMessage, serverCmdProtoMap)
		if protoData == nil {
			logger.Error("encode proto data is nil")
			return nil
		}
		kcpMsg.ProtoData = protoData
	} else {
		kcpMsg.ProtoData = nil
	}
	if config.GetConfig().Hk4e.ClientProtoProxyEnable {
		serverCmdId := kcpMsg.CmdId
		serverProtoData := kcpMsg.ProtoData
		serverProtoObj := serverCmdProtoMap.GetProtoObjByCmdId(serverCmdId)
		if serverProtoObj == nil {
			logger.Error("get server proto obj is nil, serverCmdId: %v", serverCmdId)
			return nil
		}
		err := pb.Unmarshal(serverProtoData, serverProtoObj)
		if err != nil {
			logger.Error("unmarshal server proto error: %v", err)
			return nil
		}
		ConvServerPbDataToClient(serverProtoObj, clientCmdProtoMap)
		cmdName := serverCmdProtoMap.GetCmdNameByCmdId(serverCmdId)
		if cmdName == "" {
			logger.Error("get cmdName is nil, serverCmdId: %v", serverCmdId)
			return nil
		}
		clientProtoObj := GetClientProtoObjByName(cmdName, clientCmdProtoMap)
		if clientProtoObj == nil {
			logger.Error("get client proto obj is nil, cmdName: %v", cmdName)
			return nil
		}
		err = object.CopyProtoBufSameField(clientProtoObj, serverProtoObj)
		if err != nil {
			logger.Error("copy proto obj error: %v", err)
			return nil
		}
		clientProtoData, err := pb.Marshal(clientProtoObj)
		if err != nil {
			logger.Error("marshal client proto error: %v", err)
			return nil
		}
		clientCmdId := clientCmdProtoMap.GetClientCmdIdByCmdName(cmdName)
		if clientCmdId == 0 {
			logger.Error("get client cmdId is nil, cmdName: %v", cmdName)
			return nil
		}
		kcpMsg.CmdId = clientCmdId
		kcpMsg.ProtoData = clientProtoData
	}
	return kcpMsg
}

func DecodePayloadToProto(cmdId uint16, protoData []byte, serverCmdProtoMap *cmd.CmdProtoMap) (protoObj pb.Message) {
	protoObj = serverCmdProtoMap.GetProtoObjCacheByCmdId(cmdId)
	if protoObj == nil {
		logger.Error("get new proto object is nil")
		return nil
	}
	err := pb.Unmarshal(protoData, protoObj)
	if err != nil {
		logger.Error("unmarshal proto data err: %v", err)
		return nil
	}
	return protoObj
}

func EncodeProtoToPayload(protoObj pb.Message, serverCmdProtoMap *cmd.CmdProtoMap) (protoData []byte) {
	var err error = nil
	protoData, err = pb.Marshal(protoObj)
	if err != nil {
		logger.Error("marshal proto object err: %v", err)
		return nil
	}
	return protoData
}

// 网关客户端协议代理相关反射方法

func GetClientProtoObjByName(protoObjName string, clientCmdProtoMap *client_proto.ClientCmdProtoMap) pb.Message {
	fn := clientCmdProtoMap.RefValue.MethodByName("GetClientProtoObjByName")
	if !fn.IsValid() {
		logger.Error("fn is nil")
		return nil
	}
	ret := fn.Call([]reflect.Value{reflect.ValueOf(protoObjName)})
	obj := ret[0].Interface()
	if obj == nil {
		logger.Error("try to get a not exist proto obj, protoObjName: %v", protoObjName)
		return nil
	}
	clientProtoObj := obj.(pb.Message)
	return clientProtoObj
}

// 网关客户端协议代理二级pb数据转换

const (
	ClientPbDataToServer = iota
	ServerPbDataToClient
)

func ConvClientPbDataToServer(protoObj pb.Message, clientCmdProtoMap *client_proto.ClientCmdProtoMap) pb.Message {
	cmdName := string(protoObj.ProtoReflect().Descriptor().FullName())
	if strings.Contains(cmdName, "proto.") {
		cmdName = strings.Split(cmdName, ".")[1]
	}
	switch cmdName {
	case "CombatInvocationsNotify":
		ntf := protoObj.(*proto.CombatInvocationsNotify)
		for _, entry := range ntf.InvokeList {
			HandleCombatInvokeEntry(ClientPbDataToServer, entry, clientCmdProtoMap)
		}
	case "AbilityInvocationsNotify":
		ntf := protoObj.(*proto.AbilityInvocationsNotify)
		for _, entry := range ntf.Invokes {
			HandleAbilityInvokeEntry(ClientPbDataToServer, entry, clientCmdProtoMap)
		}
	case "ClientAbilityInitFinishNotify":
		ntf := protoObj.(*proto.ClientAbilityInitFinishNotify)
		for _, entry := range ntf.Invokes {
			HandleAbilityInvokeEntry(ClientPbDataToServer, entry, clientCmdProtoMap)
		}
	case "ClientAbilityChangeNotify":
		ntf := protoObj.(*proto.ClientAbilityChangeNotify)
		for _, entry := range ntf.Invokes {
			HandleAbilityInvokeEntry(ClientPbDataToServer, entry, clientCmdProtoMap)
		}
	}
	return protoObj
}

func ConvServerPbDataToClient(protoObj pb.Message, clientCmdProtoMap *client_proto.ClientCmdProtoMap) pb.Message {
	cmdName := string(protoObj.ProtoReflect().Descriptor().FullName())
	if strings.Contains(cmdName, "proto.") {
		cmdName = strings.Split(cmdName, ".")[1]
	}
	switch cmdName {
	case "CombatInvocationsNotify":
		ntf := protoObj.(*proto.CombatInvocationsNotify)
		for _, entry := range ntf.InvokeList {
			HandleCombatInvokeEntry(ServerPbDataToClient, entry, clientCmdProtoMap)
		}
	case "AbilityInvocationsNotify":
		ntf := protoObj.(*proto.AbilityInvocationsNotify)
		for _, entry := range ntf.Invokes {
			HandleAbilityInvokeEntry(ServerPbDataToClient, entry, clientCmdProtoMap)
		}
	case "ClientAbilityInitFinishNotify":
		ntf := protoObj.(*proto.ClientAbilityInitFinishNotify)
		for _, entry := range ntf.Invokes {
			HandleAbilityInvokeEntry(ServerPbDataToClient, entry, clientCmdProtoMap)
		}
	case "ClientAbilityChangeNotify":
		ntf := protoObj.(*proto.ClientAbilityChangeNotify)
		for _, entry := range ntf.Invokes {
			HandleAbilityInvokeEntry(ServerPbDataToClient, entry, clientCmdProtoMap)
		}
	}
	return protoObj
}

func ConvClientServerPbData(convType int, protoObjName string, serverProtoObj pb.Message, protoDataRef *[]byte,
	clientCmdProtoMap *client_proto.ClientCmdProtoMap) {
	switch convType {
	case ClientPbDataToServer:
		clientProtoObj := GetClientProtoObjByName(protoObjName, clientCmdProtoMap)
		if clientProtoObj == nil {
			return
		}
		err := pb.Unmarshal(*protoDataRef, clientProtoObj)
		if err != nil {
			return
		}
		err = object.CopyProtoBufSameField(serverProtoObj, clientProtoObj)
		if err != nil {
			return
		}
		serverProtoData, err := pb.Marshal(serverProtoObj)
		if err != nil {
			return
		}
		*protoDataRef = serverProtoData
	case ServerPbDataToClient:
		err := pb.Unmarshal(*protoDataRef, serverProtoObj)
		if err != nil {
			return
		}
		clientProtoObj := GetClientProtoObjByName(protoObjName, clientCmdProtoMap)
		if clientProtoObj == nil {
			return
		}
		err = object.CopyProtoBufSameField(clientProtoObj, serverProtoObj)
		if err != nil {
			return
		}
		clientProtoData, err := pb.Marshal(clientProtoObj)
		if err != nil {
			return
		}
		*protoDataRef = clientProtoData
	}
}

func HandleCombatInvokeEntry(convType int, entry *proto.CombatInvokeEntry, clientCmdProtoMap *client_proto.ClientCmdProtoMap) {
	switch entry.ArgumentType {
	case proto.CombatTypeArgument_COMBAT_EVT_BEING_HIT:
		ConvClientServerPbData(convType, "EvtBeingHitInfo", new(proto.EvtBeingHitInfo), &entry.CombatData, clientCmdProtoMap)
	case proto.CombatTypeArgument_COMBAT_ANIMATOR_STATE_CHANGED:
		ConvClientServerPbData(convType, "EvtAnimatorStateChangedInfo", new(proto.EvtAnimatorStateChangedInfo), &entry.CombatData, clientCmdProtoMap)
	case proto.CombatTypeArgument_COMBAT_FACE_TO_DIR:
		ConvClientServerPbData(convType, "EvtFaceToDirInfo", new(proto.EvtFaceToDirInfo), &entry.CombatData, clientCmdProtoMap)
	case proto.CombatTypeArgument_COMBAT_SET_ATTACK_TARGET:
		ConvClientServerPbData(convType, "EvtSetAttackTargetInfo", new(proto.EvtSetAttackTargetInfo), &entry.CombatData, clientCmdProtoMap)
	case proto.CombatTypeArgument_COMBAT_RUSH_MOVE:
		ConvClientServerPbData(convType, "EvtRushMoveInfo", new(proto.EvtRushMoveInfo), &entry.CombatData, clientCmdProtoMap)
	case proto.CombatTypeArgument_COMBAT_ANIMATOR_PARAMETER_CHANGED:
		ConvClientServerPbData(convType, "EvtAnimatorParameterInfo", new(proto.EvtAnimatorParameterInfo), &entry.CombatData, clientCmdProtoMap)
	case proto.CombatTypeArgument_ENTITY_MOVE:
		ConvClientServerPbData(convType, "EntityMoveInfo", new(proto.EntityMoveInfo), &entry.CombatData, clientCmdProtoMap)
	case proto.CombatTypeArgument_SYNC_ENTITY_POSITION:
		ConvClientServerPbData(convType, "EvtSyncEntityPositionInfo", new(proto.EvtSyncEntityPositionInfo), &entry.CombatData, clientCmdProtoMap)
	case proto.CombatTypeArgument_COMBAT_STEER_MOTION_INFO:
		ConvClientServerPbData(convType, "EvtCombatSteerMotionInfo", new(proto.EvtCombatSteerMotionInfo), &entry.CombatData, clientCmdProtoMap)
	case proto.CombatTypeArgument_COMBAT_FORCE_SET_POS_INFO:
		ConvClientServerPbData(convType, "EvtCombatForceSetPosInfo", new(proto.EvtCombatForceSetPosInfo), &entry.CombatData, clientCmdProtoMap)
	case proto.CombatTypeArgument_COMBAT_COMPENSATE_POS_DIFF:
		ConvClientServerPbData(convType, "EvtCompensatePosDiffInfo", new(proto.EvtCompensatePosDiffInfo), &entry.CombatData, clientCmdProtoMap)
	case proto.CombatTypeArgument_COMBAT_MONSTER_DO_BLINK:
		ConvClientServerPbData(convType, "EvtMonsterDoBlink", new(proto.EvtMonsterDoBlink), &entry.CombatData, clientCmdProtoMap)
	case proto.CombatTypeArgument_COMBAT_FIXED_RUSH_MOVE:
		ConvClientServerPbData(convType, "EvtFixedRushMove", new(proto.EvtFixedRushMove), &entry.CombatData, clientCmdProtoMap)
	case proto.CombatTypeArgument_COMBAT_SYNC_TRANSFORM:
		ConvClientServerPbData(convType, "EvtSyncTransform", new(proto.EvtSyncTransform), &entry.CombatData, clientCmdProtoMap)
	case proto.CombatTypeArgument_COMBAT_LIGHT_CORE_MOVE:
		ConvClientServerPbData(convType, "EvtLightCoreMove", new(proto.EvtLightCoreMove), &entry.CombatData, clientCmdProtoMap)
	case proto.CombatTypeArgument_COMBAT_BEING_HEALED_NTF:
		ConvClientServerPbData(convType, "EvtBeingHealedNotify", new(proto.EvtBeingHealedNotify), &entry.CombatData, clientCmdProtoMap)
	case proto.CombatTypeArgument_COMBAT_SKILL_ANCHOR_POSITION_NTF:
		ConvClientServerPbData(convType, "EvtSyncSkillAnchorPosition", new(proto.EvtSyncSkillAnchorPosition), &entry.CombatData, clientCmdProtoMap)
	case proto.CombatTypeArgument_COMBAT_GRAPPLING_HOOK_MOVE:
		ConvClientServerPbData(convType, "EvtGrapplingHookMove", new(proto.EvtGrapplingHookMove), &entry.CombatData, clientCmdProtoMap)
	}
}

func HandleAbilityInvokeEntry(convType int, entry *proto.AbilityInvokeEntry, clientCmdProtoMap *client_proto.ClientCmdProtoMap) {
	switch entry.ArgumentType {
	case proto.AbilityInvokeArgument_ABILITY_META_MODIFIER_CHANGE:
		ConvClientServerPbData(convType, "AbilityMetaModifierChange", new(proto.AbilityMetaModifierChange), &entry.AbilityData, clientCmdProtoMap)
	case proto.AbilityInvokeArgument_ABILITY_META_SPECIAL_FLOAT_ARGUMENT:
		ConvClientServerPbData(convType, "AbilityMetaSpecialFloatArgument", new(proto.AbilityMetaSpecialFloatArgument), &entry.AbilityData, clientCmdProtoMap)
	case proto.AbilityInvokeArgument_ABILITY_META_OVERRIDE_PARAM:
		ConvClientServerPbData(convType, "AbilityScalarValueEntry", new(proto.AbilityScalarValueEntry), &entry.AbilityData, clientCmdProtoMap)
	case proto.AbilityInvokeArgument_ABILITY_META_CLEAR_OVERRIDE_PARAM:
		ConvClientServerPbData(convType, "AbilityString", new(proto.AbilityString), &entry.AbilityData, clientCmdProtoMap)
	case proto.AbilityInvokeArgument_ABILITY_META_REINIT_OVERRIDEMAP:
		ConvClientServerPbData(convType, "AbilityMetaReInitOverrideMap", new(proto.AbilityMetaReInitOverrideMap), &entry.AbilityData, clientCmdProtoMap)
	case proto.AbilityInvokeArgument_ABILITY_META_GLOBAL_FLOAT_VALUE:
		ConvClientServerPbData(convType, "AbilityScalarValueEntry", new(proto.AbilityScalarValueEntry), &entry.AbilityData, clientCmdProtoMap)
	case proto.AbilityInvokeArgument_ABILITY_META_CLEAR_GLOBAL_FLOAT_VALUE:
		ConvClientServerPbData(convType, "AbilityString", new(proto.AbilityString), &entry.AbilityData, clientCmdProtoMap)
	case proto.AbilityInvokeArgument_ABILITY_META_ABILITY_ELEMENT_STRENGTH:
		ConvClientServerPbData(convType, "AbilityFloatValue", new(proto.AbilityFloatValue), &entry.AbilityData, clientCmdProtoMap)
	case proto.AbilityInvokeArgument_ABILITY_META_ADD_OR_GET_ABILITY_AND_TRIGGER:
		ConvClientServerPbData(convType, "AbilityMetaAddOrGetAbilityAndTrigger", new(proto.AbilityMetaAddOrGetAbilityAndTrigger), &entry.AbilityData, clientCmdProtoMap)
	case proto.AbilityInvokeArgument_ABILITY_META_SET_KILLED_SETATE:
		ConvClientServerPbData(convType, "AbilityMetaSetKilledState", new(proto.AbilityMetaSetKilledState), &entry.AbilityData, clientCmdProtoMap)
	case proto.AbilityInvokeArgument_ABILITY_META_SET_ABILITY_TRIGGER:
		ConvClientServerPbData(convType, "AbilityMetaSetAbilityTrigger", new(proto.AbilityMetaSetAbilityTrigger), &entry.AbilityData, clientCmdProtoMap)
	case proto.AbilityInvokeArgument_ABILITY_META_ADD_NEW_ABILITY:
		ConvClientServerPbData(convType, "AbilityMetaAddAbility", new(proto.AbilityMetaAddAbility), &entry.AbilityData, clientCmdProtoMap)
	case proto.AbilityInvokeArgument_ABILITY_META_SET_MODIFIER_APPLY_ENTITY:
		ConvClientServerPbData(convType, "AbilityMetaSetModifierApplyEntityId", new(proto.AbilityMetaSetModifierApplyEntityId), &entry.AbilityData, clientCmdProtoMap)
	case proto.AbilityInvokeArgument_ABILITY_META_MODIFIER_DURABILITY_CHANGE:
		ConvClientServerPbData(convType, "AbilityMetaModifierDurabilityChange", new(proto.AbilityMetaModifierDurabilityChange), &entry.AbilityData, clientCmdProtoMap)
	case proto.AbilityInvokeArgument_ABILITY_META_ELEMENT_REACTION_VISUAL:
		ConvClientServerPbData(convType, "AbilityMetaElementReactionVisual", new(proto.AbilityMetaElementReactionVisual), &entry.AbilityData, clientCmdProtoMap)
	case proto.AbilityInvokeArgument_ABILITY_META_SET_POSE_PARAMETER:
		ConvClientServerPbData(convType, "AbilityMetaSetPoseParameter", new(proto.AbilityMetaSetPoseParameter), &entry.AbilityData, clientCmdProtoMap)
	case proto.AbilityInvokeArgument_ABILITY_META_UPDATE_BASE_REACTION_DAMAGE:
		ConvClientServerPbData(convType, "AbilityMetaUpdateBaseReactionDamage", new(proto.AbilityMetaUpdateBaseReactionDamage), &entry.AbilityData, clientCmdProtoMap)
	case proto.AbilityInvokeArgument_ABILITY_META_TRIGGER_ELEMENT_REACTION:
		ConvClientServerPbData(convType, "AbilityMetaTriggerElementReaction", new(proto.AbilityMetaTriggerElementReaction), &entry.AbilityData, clientCmdProtoMap)
	case proto.AbilityInvokeArgument_ABILITY_META_LOSE_HP:
		ConvClientServerPbData(convType, "AbilityMetaLoseHp", new(proto.AbilityMetaLoseHp), &entry.AbilityData, clientCmdProtoMap)
	case proto.AbilityInvokeArgument_ABILITY_META_DURABILITY_IS_ZERO:
		ConvClientServerPbData(convType, "AbilityMetaDurabilityIsZero", new(proto.AbilityMetaDurabilityIsZero), &entry.AbilityData, clientCmdProtoMap)
	case proto.AbilityInvokeArgument_ABILITY_ACTION_TRIGGER_ABILITY:
		ConvClientServerPbData(convType, "AbilityActionTriggerAbility", new(proto.AbilityActionTriggerAbility), &entry.AbilityData, clientCmdProtoMap)
	case proto.AbilityInvokeArgument_ABILITY_ACTION_SET_CRASH_DAMAGE:
		ConvClientServerPbData(convType, "AbilityActionSetCrashDamage", new(proto.AbilityActionSetCrashDamage), &entry.AbilityData, clientCmdProtoMap)
	case proto.AbilityInvokeArgument_ABILITY_ACTION_SUMMON:
		ConvClientServerPbData(convType, "AbilityActionSummon", new(proto.AbilityActionSummon), &entry.AbilityData, clientCmdProtoMap)
	case proto.AbilityInvokeArgument_ABILITY_ACTION_BLINK:
		ConvClientServerPbData(convType, "AbilityActionBlink", new(proto.AbilityActionBlink), &entry.AbilityData, clientCmdProtoMap)
	case proto.AbilityInvokeArgument_ABILITY_ACTION_CREATE_GADGET:
		ConvClientServerPbData(convType, "AbilityActionCreateGadget", new(proto.AbilityActionCreateGadget), &entry.AbilityData, clientCmdProtoMap)
	case proto.AbilityInvokeArgument_ABILITY_ACTION_APPLY_LEVEL_MODIFIER:
		ConvClientServerPbData(convType, "AbilityApplyLevelModifier", new(proto.AbilityApplyLevelModifier), &entry.AbilityData, clientCmdProtoMap)
	case proto.AbilityInvokeArgument_ABILITY_ACTION_GENERATE_ELEM_BALL:
		ConvClientServerPbData(convType, "AbilityActionGenerateElemBall", new(proto.AbilityActionGenerateElemBall), &entry.AbilityData, clientCmdProtoMap)
	case proto.AbilityInvokeArgument_ABILITY_ACTION_SET_RANDOM_OVERRIDE_MAP_VALUE:
		ConvClientServerPbData(convType, "AbilityActionSetRandomOverrideMapValue", new(proto.AbilityActionSetRandomOverrideMapValue), &entry.AbilityData, clientCmdProtoMap)
	case proto.AbilityInvokeArgument_ABILITY_ACTION_SERVER_MONSTER_LOG:
		ConvClientServerPbData(convType, "AbilityActionServerMonsterLog", new(proto.AbilityActionServerMonsterLog), &entry.AbilityData, clientCmdProtoMap)
	case proto.AbilityInvokeArgument_ABILITY_ACTION_CREATE_TILE:
		ConvClientServerPbData(convType, "AbilityActionCreateTile", new(proto.AbilityActionCreateTile), &entry.AbilityData, clientCmdProtoMap)
	case proto.AbilityInvokeArgument_ABILITY_ACTION_DESTROY_TILE:
		ConvClientServerPbData(convType, "AbilityActionDestroyTile", new(proto.AbilityActionDestroyTile), &entry.AbilityData, clientCmdProtoMap)
	case proto.AbilityInvokeArgument_ABILITY_ACTION_FIRE_AFTER_IMAGE:
		ConvClientServerPbData(convType, "AbilityActionFireAfterImgae", new(proto.AbilityActionFireAfterImgae), &entry.AbilityData, clientCmdProtoMap)
	case proto.AbilityInvokeArgument_ABILITY_ACTION_DEDUCT_STAMINA:
		ConvClientServerPbData(convType, "AbilityActionDeductStamina", new(proto.AbilityActionDeductStamina), &entry.AbilityData, clientCmdProtoMap)
	case proto.AbilityInvokeArgument_ABILITY_ACTION_HIT_EFFECT:
		ConvClientServerPbData(convType, "AbilityActionHitEffect", new(proto.AbilityActionHitEffect), &entry.AbilityData, clientCmdProtoMap)
	case proto.AbilityInvokeArgument_ABILITY_ACTION_SET_BULLET_TRACK_TARGET:
		ConvClientServerPbData(convType, "AbilityActionSetBulletTrackTarget", new(proto.AbilityActionSetBulletTrackTarget), &entry.AbilityData, clientCmdProtoMap)
	case proto.AbilityInvokeArgument_ABILITY_MIXIN_AVATAR_STEER_BY_CAMERA:
		ConvClientServerPbData(convType, "AbilityMixinAvatarSteerByCamera", new(proto.AbilityMixinAvatarSteerByCamera), &entry.AbilityData, clientCmdProtoMap)
	case proto.AbilityInvokeArgument_ABILITY_MIXIN_WIND_ZONE:
		ConvClientServerPbData(convType, "AbilityMixinWindZone", new(proto.AbilityMixinWindZone), &entry.AbilityData, clientCmdProtoMap)
	case proto.AbilityInvokeArgument_ABILITY_MIXIN_COST_STAMINA:
		ConvClientServerPbData(convType, "AbilityMixinCostStamina", new(proto.AbilityMixinCostStamina), &entry.AbilityData, clientCmdProtoMap)
	case proto.AbilityInvokeArgument_ABILITY_MIXIN_ELEMENT_SHIELD:
		ConvClientServerPbData(convType, "AbilityMixinElementShield", new(proto.AbilityMixinElementShield), &entry.AbilityData, clientCmdProtoMap)
	case proto.AbilityInvokeArgument_ABILITY_MIXIN_GLOBAL_SHIELD:
		ConvClientServerPbData(convType, "AbilityMixinGlobalShield", new(proto.AbilityMixinGlobalShield), &entry.AbilityData, clientCmdProtoMap)
	case proto.AbilityInvokeArgument_ABILITY_MIXIN_SHIELD_BAR:
		ConvClientServerPbData(convType, "AbilityMixinShieldBar", new(proto.AbilityMixinShieldBar), &entry.AbilityData, clientCmdProtoMap)
	case proto.AbilityInvokeArgument_ABILITY_MIXIN_WIND_SEED_SPAWNER:
		ConvClientServerPbData(convType, "AbilityMixinWindSeedSpawner", new(proto.AbilityMixinWindSeedSpawner), &entry.AbilityData, clientCmdProtoMap)
	case proto.AbilityInvokeArgument_ABILITY_MIXIN_DO_ACTION_BY_ELEMENT_REACTION:
		ConvClientServerPbData(convType, "AbilityMixinDoActionByElementReaction", new(proto.AbilityMixinDoActionByElementReaction), &entry.AbilityData, clientCmdProtoMap)
	case proto.AbilityInvokeArgument_ABILITY_MIXIN_FIELD_ENTITY_COUNT_CHANGE:
		ConvClientServerPbData(convType, "AbilityMixinFieldEntityCountChange", new(proto.AbilityMixinFieldEntityCountChange), &entry.AbilityData, clientCmdProtoMap)
	case proto.AbilityInvokeArgument_ABILITY_MIXIN_SCENE_PROP_SYNC:
		ConvClientServerPbData(convType, "AbilityMixinScenePropSync", new(proto.AbilityMixinScenePropSync), &entry.AbilityData, clientCmdProtoMap)
	case proto.AbilityInvokeArgument_ABILITY_MIXIN_WIDGET_MP_SUPPORT:
		ConvClientServerPbData(convType, "AbilityMixinWidgetMpSupport", new(proto.AbilityMixinWidgetMpSupport), &entry.AbilityData, clientCmdProtoMap)
	case proto.AbilityInvokeArgument_ABILITY_MIXIN_DO_ACTION_BY_SELF_MODIFIER_ELEMENT_DURABILITY_RATIO:
		ConvClientServerPbData(convType, "AbilityMixinDoActionBySelfModifierElementDurabilityRatio", new(proto.AbilityMixinDoActionBySelfModifierElementDurabilityRatio), &entry.AbilityData, clientCmdProtoMap)
	case proto.AbilityInvokeArgument_ABILITY_MIXIN_FIREWORKS_LAUNCHER:
		ConvClientServerPbData(convType, "AbilityMixinFireworksLauncher", new(proto.AbilityMixinFireworksLauncher), &entry.AbilityData, clientCmdProtoMap)
	case proto.AbilityInvokeArgument_ABILITY_MIXIN_ATTACK_RESULT_CREATE_COUNT:
		ConvClientServerPbData(convType, "AttackResultCreateCount", new(proto.AttackResultCreateCount), &entry.AbilityData, clientCmdProtoMap)
	case proto.AbilityInvokeArgument_ABILITY_MIXIN_UGC_TIME_CONTROL:
		ConvClientServerPbData(convType, "AbilityMixinUGCTimeControl", new(proto.AbilityMixinUGCTimeControl), &entry.AbilityData, clientCmdProtoMap)
	case proto.AbilityInvokeArgument_ABILITY_MIXIN_AVATAR_COMBAT:
		ConvClientServerPbData(convType, "AbilityMixinAvatarCombat", new(proto.AbilityMixinAvatarCombat), &entry.AbilityData, clientCmdProtoMap)
	case proto.AbilityInvokeArgument_ABILITY_MIXIN_UI_INTERACT:
		ConvClientServerPbData(convType, "AbilityMixinUIInteract", new(proto.AbilityMixinUIInteract), &entry.AbilityData, clientCmdProtoMap)
	case proto.AbilityInvokeArgument_ABILITY_MIXIN_SHOOT_FROM_CAMERA:
		ConvClientServerPbData(convType, "AbilityMixinShootFromCamera", new(proto.AbilityMixinShootFromCamera), &entry.AbilityData, clientCmdProtoMap)
	case proto.AbilityInvokeArgument_ABILITY_MIXIN_ERASE_BRICK_ACTIVITY:
		ConvClientServerPbData(convType, "AbilityMixinEraseBrickActivity", new(proto.AbilityMixinEraseBrickActivity), &entry.AbilityData, clientCmdProtoMap)
	case proto.AbilityInvokeArgument_ABILITY_MIXIN_BREAKOUT:
		ConvClientServerPbData(convType, "AbilityMixinBreakout", new(proto.AbilityMixinBreakout), &entry.AbilityData, clientCmdProtoMap)
	}
}
