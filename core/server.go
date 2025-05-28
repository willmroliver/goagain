package core

import (
	"context"
)

type Server interface {
	Run(context.Context)
	Accept() (Conn, error)
	Close(Conn) error
}
