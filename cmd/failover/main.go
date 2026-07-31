package main

import (
	"context"
	"log"

	"dns-failover/internal/config"
	"dns-failover/internal/dns"
	"dns-failover/internal/failover"
	"dns-failover/internal/logger"
	"dns-failover/internal/monitor"
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


	logg := logger.New(
		cfg.Log.Level,
		cfg.Log.Format,
	)


	dnsProvider := dns.NewSelectel(
		cfg.Selectel.AccountID,
		cfg.Selectel.ProjectName,
		cfg.Selectel.Username,
		cfg.Selectel.Password,
	)


	// Конвертация config.Support -> notifier.Support
	support := notifier.Support{
		Enabled: cfg.Support.Enabled,
	}


	for _, link := range cfg.Support.Links {

		support.Links = append(
			support.Links,
			notifier.SupportLink{
				Title: link.Title,
				URL:   link.URL,
			},
		)
	}


	telegram := notifier.NewTelegram(
		cfg.Telegram.Token,
		cfg.Telegram.ChatID,
		"internal/notifier/images",
		support,
	)


	fail := &failover.Failover{
		Provider: dnsProvider,
		Notifier: telegram,
	}


	m := monitor.New(
		ctx,
		cfg,
		logg,
		fail,
	)


	m.Run(ctx)
}