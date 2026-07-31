package main

import (
    "context"
    "fmt"
    "log"

    "dns-failover/internal/config"
    "dns-failover/internal/dns"
    "dns-failover/internal/notifier"
	"dns-failover/internal/failover"
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


	host := cfg.Hosts[0]

	records, err := provider.GetRecords(
		ctx,
		host.DNS.Zone,
		host.DNS.Record,
	)
	if err != nil {
		log.Fatal(err)
	}

	for i := range records {
		for _, cfg := range host.DNS.Records {
			if cfg.IP == records[i].IP {
				records[i].Country = cfg.Country
				records[i].Priority = cfg.Priority
				break
			}
		}
	}
	
	fmt.Println("Current records:")
	for _, r := range records {
		fmt.Printf(
			"ID=%s TTL=%d IP=%s disabled=%v\n",
			r.ID,
			r.TTL,
			r.IP,
			r.Disabled,
		)
	}

	// Для теста меняем состояние двух записей местами
	if len(records) >= 2 {
		records[0].Disabled = true
		records[1].Disabled = false
	} else {
		log.Fatal("need at least two records")
	}

	err = provider.UpdateRecords(
		ctx,
		host.DNS.Zone,
		records,
	)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("DNS updated successfully!")
	telegram := notifier.NewTelegram(
		cfg.Telegram.Token,
		cfg.Telegram.ChatID,
	)

	fo := failover.Failover{
		Provider: provider,
		Telegram: telegram,
	}

	err = fo.Switch(
		ctx,
		host,
		records,
	)

	if err != nil {
		log.Fatal(err)
	}

}