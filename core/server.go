package core

import (
	"context"
)

type Server interface {
	Run(context.Context)
	NewCxn() *Cxn
}
