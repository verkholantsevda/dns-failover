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

	cfg, err := config.Load("configs/config.yaml")
	if err != nil {
		log.Fatal(err)
	}

	logg := logger.New(cfg.Log.Level, cfg.Log.Format)

	dnsProvider := dns.NewSelectel(
		cfg.Selectel.AccountID,
		cfg.Selectel.ProjectName,
		cfg.Selectel.Username,
		cfg.Selectel.Password,
	)

	support := notifier.Support{
		Enabled: cfg.Support.Enabled,
	}
	for _, link := range cfg.Support.Links {
		support.Links = append(support.Links, notifier.SupportLink{
			Title: link.Title,
			URL:   link.URL,
		})
	}

	var notifiers []notifier.Notifier

	// Telegram – включается через telegram.enabled
	if cfg.Telegram.Enabled {
		telegram := notifier.NewTelegram(
			cfg.Telegram.Token,
			cfg.Telegram.ChatID,
			"internal/notifier/images",
			support,
		)
		notifiers = append(notifiers, telegram)
	}

	// Ntfy – включается через ntfy.enabled
	if cfg.NtfyConfig.Enabled {
		ntfy := notifier.New(
			cfg.NtfyConfig.URL,
			cfg.NtfyConfig.Topic,
			cfg.NtfyConfig.Token,
		)
		notifiers = append(notifiers, ntfy)
	}

	// Композитный нотификатор
	multiNotifier := notifier.NewMultiNotifier(notifiers...)

	fail := &failover.Failover{
		Provider: dnsProvider,
		Notifier: multiNotifier,
		Hosts:    cfg.Hosts,
		Log:      logg,
	}

	m := monitor.New(ctx, cfg, logg, fail, multiNotifier)
	m.Run(ctx)
}