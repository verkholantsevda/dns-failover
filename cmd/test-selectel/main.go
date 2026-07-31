package main

import (
	"context"
	"fmt"
	"log"

	"dns-failover/internal/config"
	"dns-failover/internal/dns"
	"dns-failover/internal/failover"
	"dns-failover/internal/notifier"
)

func main() {

	ctx := context.Background()


	cfg, err := config.Load(
		"configs/config.yaml",
	)

	if err != nil {
		log.Fatal(err)
	}


	provider := dns.NewSelectel(
		cfg.Selectel.AccountID,
		cfg.Selectel.ProjectName,
		cfg.Selectel.Username,
		cfg.Selectel.Password,
	)

	support := notifier.Support{

		Enabled: cfg.Support.Enabled,

	}

	for _, link := range cfg.Support.Links {

		support.Links = append(
			support.Links,
			notifier.SupportLink{
				Title: link.Title,
				URL: link.URL,
			},
		)

	}
	telegram := notifier.NewTelegram(
		cfg.Telegram.Token,
		cfg.Telegram.ChatID,
		"internal/notifier/images",
		support,
	)


	fo := failover.Failover{
		Provider: provider,
		Notifier: telegram,
	}


	host := cfg.Hosts[0]


	records, err := provider.GetRecords(
		ctx,
		host.DNS.Zone,
		host.DNS.Record,
	)

	if err != nil {
		log.Fatal(err)
	}



	// Добавляем данные из config.yaml
	for i := range records {

		for _, cfgRecord := range host.DNS.Records {

			if cfgRecord.IP == records[i].IP {

				records[i].Country = cfgRecord.Country
				records[i].Priority = cfgRecord.Priority

				break
			}
		}
	}



	fmt.Println("Current records:")

	for _, r := range records {

		fmt.Printf(
			"ID=%s TTL=%d IP=%s country=%s priority=%d disabled=%v\n",
			r.ID,
			r.TTL,
			r.IP,
			r.Country,
			r.Priority,
			r.Disabled,
		)
	}



	// ТЕСТ:
	// отключаем первый сервер
	// включаем второй

	if len(records) >= 2 {

		records[0].Disabled = true
		records[1].Disabled = false

	} else {

		log.Fatal(
			"need at least two records",
		)
	}



	err = fo.Switch(
		ctx,
		host,
		records,
	)

	if err != nil {
		log.Fatal(err)
	}



	fmt.Println(
		"Failover completed successfully!",
	)
}