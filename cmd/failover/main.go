package main

import (
	"context"
	"log"

	"dns-failover/internal/config"
	"dns-failover/internal/dns"
	"dns-failover/internal/logger"
	"dns-failover/internal/monitor"
)

func main() {

	ctx := context.Background()


	cfg, err := config.Load(
		"configs/config.yaml",
	)

	if err != nil {
		log.Fatal(err)
	}


	logger := logger.New()


	dnsProvider := dns.NewSelectel(
		cfg.Selectel.ProjectID,
		cfg.Selectel.Username,
		cfg.Selectel.Password,
	)


	m := monitor.New(
		ctx,
		cfg,
		logger,
		dnsProvider,
	)


	m.Run(ctx)

}