package ws

import (
	"context"
	"net"
	"time"

	"github.com/willmroliver/goagain/core"
)

var inc uint = 0

type ServerConfig struct {
	Path        string
	ConnBufSize uint
	ConnTimeout time.Duration
}

type Server struct {
	Port      int
	Listener  *net.TCPListener
	KeepAlive net.KeepAliveConfig
	Conf      ServerConfig
	Conns     map[uint]core.Conn
}

func NewServer(port int) (s *Server, err error) {
	s = &Server{
		Port:  port,
		Conns: make(map[uint]core.Conn),
	}

	s.Listener, err = net.ListenTCP("tcp", &net.TCPAddr{
		IP:   net.IPv6unspecified,
		Port: port,
		Zone: "",
	})

	s.KeepAlive.Enable = true
	return
}

func (s *Server) Run(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		default:
			conn, err := s.Accept()
			if err != nil {
				if ctx.Err() != nil {
					return
				}
				// @todo - handle error
				continue
			}

			go conn.Handshake()
		}
	}
}

func (s *Server) Accept() (core.Conn, error) {
	conn, err := s.Listener.AcceptTCP()
	if err != nil {
		return nil, err
	}

	inc++

	c := &Conn{
		TCPConn: conn,
		ConnID:  inc,
		Server:  s,
		Ring:    core.NewRingBuf(0x1000),
	}

	c.SetKeepAliveConfig(s.KeepAlive)
	s.Conns[inc] = c
	return c, nil
}

func (s *Server) Close(c core.Conn) error {
	delete(s.Conns, c.(*Conn).ConnID)
	return nil
}
