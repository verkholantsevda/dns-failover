package checker

import (
	"context"
	"time"

	"github.com/go-ping/ping"
)

type PingChecker struct {
	Timeout time.Duration
	Count   int
}

func NewPingChecker() *PingChecker {
	return &PingChecker{
		Timeout: 3 * time.Second,
		Count:   3,
	}
}

func (p *PingChecker) Check(ctx context.Context, host string) (bool, error) {
	pinger, err := ping.NewPinger(host)
	if err != nil {
		return false, err
	}

	pinger.SetPrivileged(false)

	pinger.Count = p.Count
	pinger.Timeout = p.Timeout

	done := make(chan error, 1)

	go func() {
		done <- pinger.Run()
	}()

	select {
	case <-ctx.Done():
		pinger.Stop()
		return false, ctx.Err()

	case err := <-done:
		if err != nil {
			return false, err
		}
	}

	stats := pinger.Statistics()

	return stats.PacketsRecv > 0, nil
}