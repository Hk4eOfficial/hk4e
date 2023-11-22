package net

import (
	"net"
	"time"

	"hk4e/gate/kcp"
)

// tcp模式连接对象兼容层

type Conn struct {
	kcpConn *kcp.UDPSession
	tcpConn *net.TCPConn
	isKcp   bool
}

func NewKcpConn(kcpConn *kcp.UDPSession) *Conn {
	r := new(Conn)
	r.kcpConn = kcpConn
	r.isKcp = true
	return r
}

func NewTcpConn(tcpConn *net.TCPConn) *Conn {
	r := new(Conn)
	r.tcpConn = tcpConn
	r.isKcp = false
	return r
}

func (c *Conn) IsTcpMode() bool {
	return !c.isKcp
}

func (c *Conn) GetSessionId() uint32 {
	if c.isKcp {
		return c.kcpConn.GetSessionId()
	} else {
		return 0
	}
}

func (c *Conn) GetConv() uint32 {
	if c.isKcp {
		return c.kcpConn.GetConv()
	} else {
		return 0
	}
}

func (c *Conn) Close() {
	if c.isKcp {
		_ = c.kcpConn.Close()
	} else {
		_ = c.tcpConn.Close()
	}
}

func (c *Conn) RemoteAddr() string {
	if c.isKcp {
		return c.kcpConn.RemoteAddr().String()
	} else {
		return c.tcpConn.RemoteAddr().String()
	}
}

func (c *Conn) SetReadDeadline(t time.Time) {
	if c.isKcp {
		_ = c.kcpConn.SetReadDeadline(t)
	} else {
		_ = c.tcpConn.SetReadDeadline(t)
	}
}

func (c *Conn) Read(b []byte) (int, error) {
	if c.isKcp {
		return c.kcpConn.Read(b)
	} else {
		return c.tcpConn.Read(b)
	}
}

func (c *Conn) SetWriteDeadline(t time.Time) {
	if c.isKcp {
		_ = c.kcpConn.SetWriteDeadline(t)
	} else {
		_ = c.tcpConn.SetWriteDeadline(t)
	}
}

func (c *Conn) Write(b []byte) (int, error) {
	if c.isKcp {
		return c.kcpConn.Write(b)
	} else {
		return c.tcpConn.Write(b)
	}
}

func (c *Conn) GetKcpRTO() uint32 {
	if c.isKcp {
		return c.kcpConn.GetRTO()
	} else {
		return 0
	}
}

func (c *Conn) GetKcpSRTT() int32 {
	if c.isKcp {
		return c.kcpConn.GetSRTT()
	} else {
		return 0
	}
}

func (c *Conn) GetKcpSRTTVar() int32 {
	if c.isKcp {
		return c.kcpConn.GetSRTTVar()
	} else {
		return 0
	}
}
