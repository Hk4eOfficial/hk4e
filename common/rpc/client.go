package rpc

import (
	"hk4e/common/config"
	gsapi "hk4e/gs/api"
	nodeapi "hk4e/node/api"

	"github.com/byebyebruce/natsrpc"
	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/encoders/protobuf"
)

// natsrpc客户端

// DiscoveryClient node的discovery服务
type DiscoveryClient struct {
	nodeapi.DiscoveryNATSRPCClient
}

func NewDiscoveryClient() (*DiscoveryClient, error) {
	conn, err := nats.Connect(config.GetConfig().MQ.NatsUrl)
	if err != nil {
		return nil, err
	}
	discoveryClient, err := newDiscoveryClient(conn)
	if err != nil {
		return nil, err
	}
	return discoveryClient, nil
}

func newDiscoveryClient(conn *nats.Conn) (*DiscoveryClient, error) {
	enc, err := nats.NewEncodedConn(conn, protobuf.PROTOBUF_ENCODER)
	if err != nil {
		return nil, err
	}
	cli, err := nodeapi.NewDiscoveryNATSRPCClient(enc)
	if err != nil {
		return nil, err
	}
	return &DiscoveryClient{
		DiscoveryNATSRPCClient: cli,
	}, nil
}

// GMClient gs的gm服务
type GMClient struct {
	gsapi.GMNATSRPCClient
}

func NewGMClient(gsId uint32) (*GMClient, error) {
	conn, err := nats.Connect(config.GetConfig().MQ.NatsUrl)
	if err != nil {
		return nil, err
	}
	gmClient, err := newGmClient(conn, gsId)
	if err != nil {
		return nil, err
	}
	return gmClient, nil
}

func newGmClient(conn *nats.Conn, gsId uint32) (*GMClient, error) {
	enc, err := nats.NewEncodedConn(conn, protobuf.PROTOBUF_ENCODER)
	if err != nil {
		return nil, err
	}
	cli, err := gsapi.NewGMNATSRPCClient(enc, natsrpc.WithClientID(gsId))
	if err != nil {
		return nil, err
	}
	return &GMClient{
		GMNATSRPCClient: cli,
	}, nil
}
