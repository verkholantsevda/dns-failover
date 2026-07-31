package monitor

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"dns-failover/internal/checker"
	"dns-failover/internal/config"
	"dns-failover/internal/dns"
	"dns-failover/internal/state"
)

type Monitor struct {
	cfg *config.Config
	log *slog.Logger

	metrics checker.MetricsChecker
	icmp    checker.ICMPChecker

	dns dns.Provider

	states map[string]*state.HostState
}

func New(
	ctx context.Context,
	cfg *config.Config,
	log *slog.Logger,
	dnsProvider dns.Provider,
) *Monitor {

	states := make(map[string]*state.HostState)
	for _, host := range cfg.Hosts {
		currentIP, err := dnsProvider.GetRecord(
			ctx,
			host.DNS.Zone,
			host.DNS.Record,
		)
		if err != nil {
			log.Error(
				"failed to get current dns record",
				"host",
				host.Name,
				"error",
				err,
			)
			// fallback из конфига
			if len(host.DNS.Records) > 0 {
				currentIP = host.DNS.Records[0].IP
			}
		}

		states[host.Name] = &state.HostState{
			ActiveIP: currentIP,
		}
		log.Info(
			"dns state initialized",
			"host",
			host.Name,
			"ip",
			currentIP,
		)
	}

	return &Monitor{
		cfg: cfg,
		log: log,
		metrics: checker.NewPrometheus(
			cfg.Prometheus.URL,
		),

		icmp: checker.NewPingChecker(),
		dns: dnsProvider,
		states: states,
	}
}

func (m *Monitor) Run(ctx context.Context) {

	ticker := time.NewTicker(
		m.cfg.Interval,
	)

	defer ticker.Stop()
	// первая проверка сразу
	m.checkHosts(ctx)
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			m.checkHosts(ctx)
		}
	}
}

func (m *Monitor) checkHosts(ctx context.Context) {
	for _, host := range m.cfg.Hosts {
		hostState := m.states[host.Name]
		query := fmt.Sprintf(
			`up{job="%s",instance="%s"}`,
			host.Prometheus.Job,
			host.Prometheus.Instance,
		)
		metricsOK, err := m.metrics.Check(
			ctx,
			query,
		)
		if err != nil {
			m.log.Error(
				"prometheus check failed",
				"host",
				host.Name,
				"error",
				err,
			)
			continue
		}
		healthy := false
		if metricsOK {
			healthy = true
			m.log.Info(
				"prometheus healthy",
				"host",
				host.Name,
			)
		} else {
			m.log.Warn(
				"prometheus failed, checking icmp",
				"host",
				host.Name,
			)
			icmpOK, err := m.icmp.Check(
				ctx,
				host.ICMPHost,
			)
			if err != nil {
				m.log.Error(
					"icmp check failed",
					"host",
					host.Name,
					"error",
					err,
				)
				continue
			}
			healthy = icmpOK
			m.log.Info(
				"icmp result",
				"host",
				host.Name,
				"healthy",
				icmpOK,
			)
		}
		if healthy {
			transition := hostState.MarkHealthy(
				m.cfg.SuccessThreshold,
			)
			switch transition {
			case state.Recovered:
				m.log.Info(
					"host recovered",
					"host",
					host.Name,
				)
				primaryIP := dns.PrimaryIP(
					host.DNS.Records,
				)
				if primaryIP != hostState.ActiveIP {
					err := m.dns.UpdateRecord(
						ctx,
						host.DNS.Zone,
						host.DNS.Record,
						primaryIP,
					)
					if err != nil {
						m.log.Error(
							"dns restore failed",
							"host",
							host.Name,
							"error",
							err,
						)
						continue
					}
					hostState.ActiveIP = primaryIP
					m.log.Info(
						"dns restored",
						"host",
						host.Name,
						"ip",
						primaryIP,
					)
				}
			}
		} else {
			transition := hostState.MarkFailed(
				m.cfg.FailThreshold,
			)
			switch transition {
			case state.Failed:
				m.log.Error(
					"host failed",
					"host",
					host.Name,
				)
				nextIP := dns.NextIP(
					host.DNS.Records,
					hostState.ActiveIP,
				)
				if nextIP == hostState.ActiveIP {
					m.log.Warn(
						"no backup ip available",
						"host",
						host.Name,
					)
					continue
				}
				err := m.dns.UpdateRecord(
					ctx,
					host.DNS.Zone,
					host.DNS.Record,
					nextIP,
				)
				if err != nil {
					m.log.Error(
						"dns switch failed",
						"host",
						host.Name,
						"error",
						err,
					)
					continue
				}
				hostState.ActiveIP = nextIP
				m.log.Info(
					"dns switched",
					"host",
					host.Name,
					"new_ip",
					nextIP,
				)
			}
		}
	}
}