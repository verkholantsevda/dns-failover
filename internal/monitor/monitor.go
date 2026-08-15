package monitor

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"dns-failover/internal/checker"
	"dns-failover/internal/config"
	"dns-failover/internal/failover"
	"dns-failover/internal/model"
	"dns-failover/internal/notifier"
	"dns-failover/internal/state"
)

type Monitor struct {
	cfg      *config.Config
	log      *slog.Logger
	metrics  checker.MetricsChecker
	icmp     checker.ICMPChecker
	failover *failover.Failover
	notifier notifier.Notifier
	states   map[string]*state.HostState
}

func New(ctx context.Context, cfg *config.Config, log *slog.Logger, failover *failover.Failover, notifier notifier.Notifier) *Monitor {
	states := make(map[string]*state.HostState)
	for _, host := range cfg.Hosts {
		states[host.Name] = &state.HostState{
			ActiveIP:     host.DNS.IP,
			LastICMPOK:   true, // по умолчанию считаем, что хост доступен
		}
		log.Info("dns state initialized", "host", host.Name, "ip", host.DNS.IP)
	}
	return &Monitor{
		cfg:      cfg,
		log:      log,
		metrics:  checker.NewPrometheus(cfg.Prometheus.URL),
		icmp:     checker.NewPingChecker(),
		failover: failover,
		notifier: notifier,
		states:   states,
	}
}

func (m *Monitor) Run(ctx context.Context) {
	m.syncDNSState(ctx)
	ticker := time.NewTicker(m.cfg.Interval)
	defer ticker.Stop()
	m.checkHosts(ctx)
	for {
		select {
		case <-ctx.Done():
			m.log.Info("monitor stopped")
			return
		case <-ticker.C:
			m.checkHosts(ctx)
		}
	}
}

// isHostHealthy проверяет доступность хоста (без изменения счётчиков)
func (m *Monitor) isHostHealthy(ctx context.Context, host model.Host) bool {
	query := fmt.Sprintf(`up{job="%s",instance="%s"}`, host.Prometheus.Job, host.Prometheus.Instance)
	metricsOK, err := m.metrics.Check(ctx, query)
	if err == nil && metricsOK {
		return true
	}
	icmpOK, err := m.icmp.Check(ctx, host.DNS.IP)
	if err != nil {
		m.log.Debug("ICMP check error during health check", "host", host.Name, "error", err)
		return false
	}
	return icmpOK
}

// syncDNSState синхронизирует состояние DNS при старте
func (m *Monitor) syncDNSState(ctx context.Context) {
	m.log.Info("starting DNS state synchronization")

	for _, host := range m.cfg.Hosts {
		hostState := m.states[host.Name]

		records, err := m.failover.Provider.GetRecords(ctx, host.DNS.Zone, host.DNS.Record)
		if err != nil {
			m.log.Error("failed to get DNS record during sync", "host", host.Name, "error", err)
			continue
		}
		if len(records) == 0 {
			m.log.Warn("no DNS record found during sync", "host", host.Name)
			continue
		}
		currentRec := records[0]

		healthy := m.isHostHealthy(ctx, host)

		if currentRec.Type == "CNAME" {
			m.log.Info("current record is CNAME", "host", host.Name, "target", currentRec.Target)
			if healthy {
				m.log.Info("host is healthy, restoring to A", "host", host.Name)
				if err := m.failover.Restore(ctx, host); err != nil {
					m.log.Error("failed to restore during sync", "host", host.Name, "error", err)
				} else {
					hostState.ActiveIP = host.DNS.IP
					hostState.FailCount = 0
					hostState.SuccessCount = 0
					hostState.Failed = false
					hostState.LastICMPOK = true
					m.log.Info("DNS restored during sync", "host", host.Name)
				}
			} else {
				m.log.Info("host is unhealthy, keeping CNAME", "host", host.Name)
			}
		} else if currentRec.Type == "A" {
			m.log.Info("current record is A", "host", host.Name, "ip", currentRec.IP)
			if !healthy {
				m.log.Info("host is unhealthy, switching to CNAME", "host", host.Name)
				if err := m.failover.Switch(ctx, host); err != nil {
					m.log.Error("failed to switch during sync", "host", host.Name, "error", err)
				} else {
					hostState.ActiveIP = ""
					hostState.FailCount = 0
					hostState.SuccessCount = 0
					hostState.Failed = true
					hostState.LastICMPOK = false
					m.log.Info("DNS switched to CNAME during sync", "host", host.Name)
				}
			} else {
				m.log.Info("host is healthy, keeping A", "host", host.Name)
			}
		} else {
			m.log.Warn("unknown record type", "host", host.Name, "type", currentRec.Type)
		}
	}

	m.log.Info("DNS state synchronization finished")
}

// checkHosts – основная периодическая проверка
func (m *Monitor) checkHosts(ctx context.Context) {
	for _, host := range m.cfg.Hosts {
		hostState := m.states[host.Name]

		query := fmt.Sprintf(`up{job="%s",instance="%s"}`, host.Prometheus.Job, host.Prometheus.Instance)
		m.log.Debug("checking prometheus", "host", host.Name, "query", query)

		metricsOK, err := m.metrics.Check(ctx, query)
		if err != nil {
			m.log.Warn("prometheus check error, falling back to ICMP", "host", host.Name, "error", err)
			metricsOK = false
		}

		// 1. Prometheus OK – сразу считаем здоровым и восстанавливаем primary
		if metricsOK {
			m.log.Info("prometheus healthy, resetting counters and restoring primary", "host", host.Name)
			hostState.FailCount = 0
			hostState.SuccessCount = 0
			hostState.Failed = false
			hostState.LastICMPOK = true

			if hostState.ActiveIP != host.DNS.IP {
				m.log.Debug("restoring primary via prometheus", "host", host.Name, "from", hostState.ActiveIP, "to", host.DNS.IP)
				if err := m.failover.Restore(ctx, host); err != nil {
					m.log.Error("failed to restore DNS", "host", host.Name, "error", err)
				} else {
					hostState.ActiveIP = host.DNS.IP
					m.log.Info("DNS restored (prometheus)", "host", host.Name, "ip", host.DNS.IP)
				}
			}
			continue
		}

		// 2. Prometheus не видит хост – проверяем ICMP по primary IP
		icmpTarget := host.DNS.IP
		m.log.Warn("prometheus failed, checking ICMP", "host", host.Name, "target", icmpTarget)

		icmpOK, err := m.icmp.Check(ctx, icmpTarget)
		if err != nil {
			m.log.Error("ICMP check failed", "host", host.Name, "error", err)
			icmpOK = false
		}

		// Отправляем уведомление только если статус ICMP изменился
		if icmpOK != hostState.LastICMPOK {
			if icmpOK {
				m.log.Info("ICMP status changed to OK", "host", host.Name)
				if err := m.notifier.SendICMPUp(ctx, host); err != nil {
					m.log.Error("failed to send ICMP UP notification", "host", host.Name, "error", err)
				}
			} else {
				m.log.Warn("ICMP status changed to FAIL", "host", host.Name)
				if err := m.notifier.SendICMPDown(ctx, host); err != nil {
					m.log.Error("failed to send ICMP DOWN notification", "host", host.Name, "error", err)
				}
			}
			hostState.LastICMPOK = icmpOK
		}

		if icmpOK {
			m.log.Info("ICMP host target OK", "host", host.Name, "target", icmpTarget)
			transition := hostState.MarkHealthy(m.cfg.SuccessThreshold)
			if transition == state.Recovered {
				m.log.Debug("restoring primary via ICMP", "host", host.Name, "from", hostState.ActiveIP, "to", host.DNS.IP)
				if err := m.failover.Restore(ctx, host); err != nil {
					m.log.Error("failed to restore DNS", "host", host.Name, "error", err)
				} else {
					hostState.ActiveIP = host.DNS.IP
					m.log.Info("DNS restored (ICMP)", "host", host.Name, "ip", host.DNS.IP)
				}
			}
			continue
		}

		// 3. ICMP FAIL – увеличиваем fail_count только если не в состоянии сбоя
		if !hostState.Failed {
			m.log.Warn("ICMP failed", "host", host.Name, "target", icmpTarget, "fail_count", hostState.FailCount+1, "threshold", m.cfg.FailThreshold)
			transition := hostState.MarkFailed(m.cfg.FailThreshold)
			if transition == state.Failed {
				m.log.Warn("host failed (ICMP)", "host", host.Name, "fail_count", hostState.FailCount)
				if err := m.failover.Switch(ctx, host); err != nil {
					m.log.Error("failed to switch DNS", "host", host.Name, "error", err)
				} else {
					hostState.ActiveIP = ""
					hostState.FailCount = 0
					hostState.SuccessCount = 0
					m.log.Warn("DNS switched to backup", "host", host.Name)
				}
			}
		} else {
			m.log.Debug("host already in failed state, skipping fail count increment", "host", host.Name)
		}
	}
}