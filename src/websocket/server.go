package websocket

import (
	"net"
	"time"
)

type ServerConfig struct {
	Path       string
	CxnBufSize uint
	CxnTimeout time.Duration
}

type Server struct {
	Port      int
	Listener  *net.TCPListener
	KeepAlive net.KeepAliveConfig
	Conf      ServerConfig
	Cxns      []*Cxn
}

func NewServer(port int) (s *Server, err error) {
	s = &Server{Port: port}

	s.Listener, err = net.ListenTCP("tcp", &net.TCPAddr{
		IP:   net.IPv6unspecified,
		Port: port,
		Zone: "",
	})

	s.KeepAlive.Enable = true

	return
}

func (s *Server) Run() {
	for {
		conn, err := s.Listener.AcceptTCP()
		if err != nil {
			// @todo - handle error
			continue
		}

		s.handleConn(conn)
	}
}

func (s *Server) handleConn(c *net.TCPConn) {
	cxn := NewCxn(s, c)
	cxn.SetKeepAliveConfig(s.KeepAlive)
	s.Cxns = append(s.Cxns, cxn)

	go cxn.Talk()
}
