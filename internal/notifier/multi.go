package notifier

import (
	"context"
	"dns-failover/internal/model"
)

type MultiNotifier struct {
	notifiers []Notifier
}

func NewMultiNotifier(notifiers ...Notifier) *MultiNotifier {
	return &MultiNotifier{notifiers: notifiers}
}

func (m *MultiNotifier) SendFailover(ctx context.Context, host, backup model.Host) error {
	for _, n := range m.notifiers {
		_ = n.SendFailover(ctx, host, backup)
	}
	return nil
}

func (m *MultiNotifier) SendRecovery(ctx context.Context, host model.Host) error {
	for _, n := range m.notifiers {
		_ = n.SendRecovery(ctx, host)
	}
	return nil
}

func (m *MultiNotifier) SendICMPDown(ctx context.Context, host model.Host) error {
	for _, n := range m.notifiers {
		_ = n.SendICMPDown(ctx, host)
	}
	return nil
}

func (m *MultiNotifier) SendICMPUp(ctx context.Context, host model.Host) error {
	for _, n := range m.notifiers {
		_ = n.SendICMPUp(ctx, host)
	}
	return nil
}