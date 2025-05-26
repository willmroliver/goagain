package websocket

import (
	"context"
	"net"
	"time"

	"github.com/willmroliver/goagain/src/container"
)

var inc uint = 0

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
	Cxns      map[uint]*Cxn
}

func NewServer(port int) (s *Server, err error) {
	s = &Server{
		Port: port,
		Cxns: make(map[uint]*Cxn),
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
			conn, err := s.Listener.AcceptTCP()
			if err != nil {
				if ctx.Err() != nil {
					return
				}
				// @todo - handle error
				continue
			}

			go s.NewCxn(conn).Talk(ctx)
		}
	}
}

func (s *Server) NewCxn(c *net.TCPConn) (cxn *Cxn) {
	inc++
	cxn = &Cxn{
		TCPConn: c,
		CxnID:   inc,
		Server:  s,
		Buf:     container.NewRing[byte](0x1000),
	}
	cxn.SetKeepAliveConfig(s.KeepAlive)
	s.Cxns[inc] = cxn
	return
}
