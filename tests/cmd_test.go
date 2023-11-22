package tests

import (
	"hk4e/protocol/cmd"
	"hk4e/protocol/proto"

	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCmd(t *testing.T) {
	t.Log("TestCmd")
	pbm := cmd.NewCmdProtoMap()
	ping_req1 := pbm.GetProtoObjByCmdId(cmd.PingReq)
	t1 := reflect.TypeOf(ping_req1)
	t.Log(t1, t1.Elem(), t1.Elem().Name())
	ping_req2 := new(proto.PingReq)
	t2 := reflect.TypeOf(ping_req2)
	t.Log(t2, t2.Elem(), t2.Elem().Name())
	assert.Equal(t, t1.Elem().Name(), t2.Elem().Name())
	assert.Equal(t, pbm.GetCmdNameByCmdId(cmd.PingReq), "PingReq")
	assert.Equal(t, int(pbm.GetCmdIdByCmdName("PingReq")), cmd.PingReq)
	assert.Equal(t, int(pbm.GetCmdIdByProtoObj(ping_req1)), cmd.PingReq)
	assert.Equal(t, int(pbm.GetCmdIdByProtoObj(ping_req2)), cmd.PingReq)
}
