package failover

import (
	"context"

	"dns-failover/internal/dns"
	"dns-failover/internal/model"
	"dns-failover/internal/notifier"
)

type Failover struct {
	Provider dns.Provider
	Notifier notifier.Notifier
}


func (f *Failover) Switch(
	ctx context.Context,
	host model.Host,
	records []model.Record,
) error {


	// Заполняем данные из config.yaml
	for i := range records {

		for _, cfg := range host.DNS.Records {

			if cfg.IP == records[i].IP {

				records[i].Country = cfg.Country
				records[i].Priority = cfg.Priority

				break
			}
		}
	}


	// Переключаем DNS
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



	return f.Notifier.SendFailover(
		ctx,
		from.Country,
		to.Country,
	)
}



func (f *Failover) Restore(
	ctx context.Context,
	host model.Host,
	records []model.Record,
) error {


	// Заполняем данные из config.yaml
	for i := range records {

		for _, cfg := range host.DNS.Records {

			if cfg.IP == records[i].IP {

				records[i].Country = cfg.Country
				records[i].Priority = cfg.Priority

				break
			}
		}
	}



	// Возвращаем основной DNS
	if err := f.Provider.UpdateRecords(
		ctx,
		host.DNS.Zone,
		records,
	); err != nil {
		return err
	}

	var to model.Record
	for _, r := range records {
		if !r.Disabled {
			to = r
			break
		}
	}

	return f.Notifier.SendRecovery(
		ctx,
		to.Country,
	)
}