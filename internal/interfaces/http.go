package interfaces

import (
	"context"
	"time"
)

type Server interface {
	Start(ctx context.Context) error
	Stop(ctx context.Context, timeout time.Duration) error
}
