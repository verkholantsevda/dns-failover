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

	logg := logger.New()

	dnsProvider := dns.NewSelectel(
		cfg.Selectel.AccountID,
		cfg.Selectel.ProjectName,
		cfg.Selectel.Username,
		cfg.Selectel.Password,
	)

	telegram := notifier.NewTelegram(
		cfg.Telegram.Token,
		cfg.Telegram.ChatID,
	)

	fail := &failover.Failover{
		Provider: dnsProvider,
		Telegram: telegram,
	}

	m := monitor.New(
		ctx,
		cfg,
		logg,
		fail,
	)

	m.Run(ctx)
}