package service

import (
	"hk4e/common/mq"
	"hk4e/node/api"
	"hk4e/node/dao"

	"github.com/byebyebruce/natsrpc"
	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/encoders/protobuf"
)

type Service struct {
	db               *dao.Dao
	discoveryService *DiscoveryService
}

func NewService(db *dao.Dao, conn *nats.Conn, messageQueue *mq.MessageQueue) (*Service, error) {
	enc, err := nats.NewEncodedConn(conn, protobuf.PROTOBUF_ENCODER)
	if err != nil {
		return nil, err
	}
	svr, err := natsrpc.NewServer(enc)
	if err != nil {
		return nil, err
	}
	discoveryService, err := NewDiscoveryService(db, messageQueue)
	if err != nil {
		return nil, err
	}
	_, err = api.RegisterDiscoveryNATSRPCServer(svr, discoveryService)
	if err != nil {
		return nil, err
	}
	s := &Service{
		db:               db,
		discoveryService: discoveryService,
	}
	return s, nil
}

func (s *Service) Close() {
	s.discoveryService.close()
}
