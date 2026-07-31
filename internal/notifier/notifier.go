package notifier

import "context"

type Notifier interface {
	Send(
		ctx context.Context,
		message string,
	) error

	SendFailover(
		ctx context.Context,
		fromCountry string,
		toCountry string,
	) error

	SendRecovery(
		ctx context.Context,
		fromCountry string,
		toCountry string,
	) error
}
