package failover

import (
	"context"
	"fmt"

	"dns-failover/internal/dns"
	"dns-failover/internal/model"
	"dns-failover/internal/notifier"
)

type Failover struct {
	Provider dns.Provider
	Telegram *notifier.Telegram
}

func (f *Failover) Switch(
	ctx context.Context,
	host model.Host,
	records []model.Record,
) error {

	// Заполняем Country и Priority из config.yaml
	for i := range records {
		for _, cfg := range host.DNS.Records {
			if cfg.IP == records[i].IP {
				records[i].Country = cfg.Country
				records[i].Priority = cfg.Priority
				break
			}
		}
	}

	// Обновляем DNS
	if err := f.Provider.UpdateRecords(
		ctx,
		host.DNS.Zone,
		records,
	); err != nil {
		return err
	}

	var from model.Record
	var to model.Record

	for _, r := range records {
		if r.Disabled {
			from = r
		} else {
			to = r
		}
	}

	fromCountry, ok := notifier.Countries[from.Country]
	if !ok {
		fromCountry = struct {
			Name string
			Flag string
		}{
			Name: from.Country,
			Flag: "🏳️",
		}
	}

	toCountry, ok := notifier.Countries[to.Country]
	if !ok {
		toCountry = struct {
			Name string
			Flag string
		}{
			Name: to.Country,
			Flag: "🏳️",
		}
	}

	message := fmt.Sprintf(
`%s WARP %s

Из-за недоступности сервера

трафик временно переключен на %s %s.

Соединение продолжает работать в штатном режиме.

Мы автоматически вернем маршрут после восстановления сервера.`,
		fromCountry.Flag,
		fromCountry.Name,
		toCountry.Flag,
		toCountry.Name,
	)

	return f.Telegram.Send(ctx, message)
}