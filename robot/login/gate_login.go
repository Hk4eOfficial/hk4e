package login

import (
	"bytes"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strconv"

	"hk4e/common/region"
	"hk4e/pkg/endec"
	"hk4e/pkg/logger"
	"hk4e/pkg/random"
	"hk4e/pkg/reflection"
	"hk4e/protocol/cmd"
	"hk4e/protocol/proto"
	"hk4e/robot/net"
)

func GateLogin(dispatchInfo *DispatchInfo, accountInfo *AccountInfo, keyId string, req *proto.GetPlayerTokenReq, clientSeq uint32) (*net.Session, error, *proto.GetPlayerTokenRsp) {
	gateAddr := dispatchInfo.GateIp + ":" + strconv.Itoa(int(dispatchInfo.GatePort))
	logger.Debug("connect gate addr: %v", gateAddr)
	session, err := net.NewSession(gateAddr, dispatchInfo.DispatchKey)
	if err != nil {
		return nil, err, nil
	}
	logger.Debug("connect gate ok")
	timeRand := random.GetTimeRand()
	clientSeedUint64 := timeRand.Uint64()
	clientSeedBuf := new(bytes.Buffer)
	err = binary.Write(clientSeedBuf, binary.BigEndian, &clientSeedUint64)
	if err != nil {
		return nil, err, nil
	}
	clientSeed := clientSeedBuf.Bytes()
	logger.Debug("clientSeed: %v, clientSeedUint64: %v", clientSeed, clientSeedUint64)
	signRsaKey, encRsaKeyMap, _ := region.LoadRegionRsaKey()
	if signRsaKey == nil || encRsaKeyMap == nil {
		return nil, errors.New("load key error"), nil
	}
	signPubkey, err := endec.RsaParsePubKeyByPrivKey(signRsaKey)
	if err != nil {
		logger.Error("parse rsa pub key error: %v", err)
		return nil, err, nil
	}
	signRsaKey, err = os.ReadFile("key/region_sign_key_pub.pem")
	if err == nil {
		logger.Debug("use pub key")
		signPubkey, err = endec.RsaParsePubKey(signRsaKey)
		if err != nil {
			logger.Error("parse rsa pub key error: %v", err)
			return nil, err, nil
		}
	}
	clientSeedEnc, err := endec.RsaEncrypt(clientSeed, signPubkey)
	if err != nil {
		logger.Error("rsa enc error: %v", err)
		return nil, err, nil
	}
	clientSeedBase64 := base64.StdEncoding.EncodeToString(clientSeedEnc)
	getPlayerTokenReq := new(proto.GetPlayerTokenReq)
	keyIdInt, err := strconv.Atoi(keyId)
	if err != nil {
		logger.Error("parse key id error: %v", err)
		return nil, err, nil
	}
	if req == nil {
		getPlayerTokenReq = &proto.GetPlayerTokenReq{
			AccountToken:  accountInfo.ComboToken,
			AccountUid:    strconv.Itoa(int(accountInfo.AccountId)),
			KeyId:         uint32(keyIdInt),
			ClientRandKey: clientSeedBase64,
			AccountType:   1,
			ChannelId:     1,
			SubChannelId:  1,
			PlatformType:  3,
		}
	} else {
		ok := reflection.CopyStructSameField(getPlayerTokenReq, req)
		if !ok {
			return nil, errors.New("copy field error"), nil
		}
		getPlayerTokenReq.KeyId = uint32(keyIdInt)
		getPlayerTokenReq.ClientRandKey = clientSeedBase64
	}
	session.SendMsgFwd(cmd.GetPlayerTokenReq, clientSeq, getPlayerTokenReq)
	getPlayerTokenReqJson, _ := json.Marshal(getPlayerTokenReq)
	logger.Debug("GetPlayerTokenReq: %v", string(getPlayerTokenReqJson))
	protoMsg := <-session.RecvChan
	if protoMsg.CmdId != cmd.GetPlayerTokenRsp {
		return nil, errors.New("recv pkt is not GetPlayerTokenRsp"), nil
	}
	getPlayerTokenRsp := protoMsg.PayloadMessage.(*proto.GetPlayerTokenRsp)
	getPlayerTokenRspJson, _ := json.Marshal(getPlayerTokenRsp)
	logger.Debug("GetPlayerTokenRsp: %v", string(getPlayerTokenRspJson))
	if getPlayerTokenRsp.Retcode != 0 {
		return nil, errors.New(fmt.Sprintf("gate login error, retCode: %v", getPlayerTokenRsp.Retcode)), nil
	}
	logger.Info("gate login ok, uid: %v", getPlayerTokenRsp.Uid)
	session.Uid = getPlayerTokenRsp.Uid
	// XOR密钥切换
	seedEnc, err := base64.StdEncoding.DecodeString(getPlayerTokenRsp.ServerRandKey)
	if err != nil {
		logger.Error("base64 decode error: %v", err)
		return nil, err, nil
	}
	encPubPrivKey, exist := encRsaKeyMap[keyId]
	if !exist {
		logger.Error("can not found key id: %v", keyId)
		return nil, err, nil
	}
	privKey, err := endec.RsaParsePrivKey(encPubPrivKey)
	if err != nil {
		logger.Error("parse rsa pub key error: %v", err)
		return nil, err, nil
	}
	seed, err := endec.RsaDecrypt(seedEnc, privKey)
	if err != nil {
		logger.Error("rsa dec error: %v", err)
		return nil, err, nil
	}
	seedUint64 := uint64(0)
	err = binary.Read(bytes.NewReader(seed), binary.BigEndian, &seedUint64)
	if err != nil {
		logger.Error("parse seed error: %v", err)
		return nil, err, nil
	}
	serverSeedUint64 := seedUint64 ^ clientSeedUint64
	logger.Debug("seed: %v, seedUint64: %v", seed, seedUint64)
	logger.Debug("serverSeedUint64: %v", serverSeedUint64)
	logger.Info("change session xor key")
	keyBlock := random.NewKeyBlock(serverSeedUint64, true)
	xorKey := keyBlock.XorKey()
	key := make([]byte, 4096)
	copy(key, xorKey[:])
	session.XorKey = key
	session.ClientVersionRandomKey = getPlayerTokenRsp.ClientVersionRandomKey
	session.SecurityCmdBuffer = getPlayerTokenRsp.SecurityCmdBuffer
	return session, nil, getPlayerTokenRsp
}
