package monitor

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"dns-failover/internal/checker"
	"dns-failover/internal/config"
	"dns-failover/internal/dns"
	"dns-failover/internal/failover"
	"dns-failover/internal/state"
)

type Monitor struct {
	cfg *config.Config
	log *slog.Logger

	metrics checker.MetricsChecker
	icmp    checker.ICMPChecker

	failover *failover.Failover

	states map[string]*state.HostState
}

func New(
	ctx context.Context,
	cfg *config.Config,
	log *slog.Logger,
	failover *failover.Failover,
) *Monitor {

	states := make(map[string]*state.HostState)

	for _, host := range cfg.Hosts {

		currentIP := ""

		if len(host.DNS.Records) > 0 {
			currentIP = host.DNS.Records[0].IP
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

		failover: failover,
		states:   states,
	}
}


func (m *Monitor) Run(ctx context.Context) {

	ticker := time.NewTicker(
		m.cfg.Interval,
	)

	defer ticker.Stop()


	m.checkHosts(ctx)


	for {

		select {

		case <-ctx.Done():

			m.log.Info(
				"monitor stopped",
			)

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


		m.log.Debug(
			"checking prometheus",
			"host",
			host.Name,
			"query",
			query,
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


		m.log.Debug(
			"prometheus result",
			"host",
			host.Name,
			"healthy",
			metricsOK,
		)


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


			m.log.Debug(
				"checking icmp",
				"host",
				host.Name,
				"target",
				host.ICMPHost,
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


			m.log.Debug(
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


			if transition != state.Recovered {
				continue
			}


			m.log.Info(
				"host recovered",
				"host",
				host.Name,
			)


			primaryIP := host.DNS.Records[0].IP


			if primaryIP == hostState.ActiveIP {
				continue
			}


			m.log.Debug(
				"preparing dns recovery",
				"host",
				host.Name,
				"from",
				hostState.ActiveIP,
				"to",
				primaryIP,
			)


			records, err := m.failover.Provider.GetRecords(
				ctx,
				host.DNS.Zone,
				host.DNS.Record,
			)


			if err != nil {

				m.log.Error(
					"get dns records failed",
					"host",
					host.Name,
					"error",
					err,
				)

				continue
			}


			for i := range records {
				records[i].Disabled = records[i].IP != primaryIP
			}


			err = m.failover.Switch(
				ctx,
				host,
				records,
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



		} else {


			transition := hostState.MarkFailed(
				m.cfg.FailThreshold,
			)


			if transition != state.Failed {
				continue
			}


			m.log.Warn(
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


			m.log.Debug(
				"preparing dns failover",
				"host",
				host.Name,
				"from",
				hostState.ActiveIP,
				"to",
				nextIP,
			)


			records, err := m.failover.Provider.GetRecords(
				ctx,
				host.DNS.Zone,
				host.DNS.Record,
			)


			if err != nil {

				m.log.Error(
					"get dns records failed",
					"host",
					host.Name,
					"error",
					err,
				)

				continue
			}


			for i := range records {
				records[i].Disabled = records[i].IP != nextIP
			}


			err = m.failover.Switch(
				ctx,
				host,
				records,
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


			m.log.Warn(
				"dns switched",
				"host",
				host.Name,
				"new_ip",
				nextIP,
			)
		}
	}
}