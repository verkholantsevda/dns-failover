package notifier

import (
	"context"
	"dns-failover/internal/model"
)

type Notifier interface {
	SendFailover(ctx context.Context, host model.Host, backup model.Host) error
	SendRecovery(ctx context.Context, host model.Host) error
	SendICMPDown(ctx context.Context, host model.Host) error
	SendICMPUp(ctx context.Context, host model.Host) error
}