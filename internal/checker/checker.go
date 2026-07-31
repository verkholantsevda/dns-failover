package checker

import "context"

type MetricsChecker interface {
	Check(ctx context.Context, query string) (bool, error)
}

type ICMPChecker interface {
	Check(ctx context.Context, host string) (bool, error)
}