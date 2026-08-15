package failover

import (
	"context"
	"fmt"
	"log/slog"

	"dns-failover/internal/dns"
	"dns-failover/internal/model"
	"dns-failover/internal/notifier"
)

type Failover struct {
	Provider dns.Provider
	Notifier notifier.Notifier
	Hosts    []model.Host
	Log      *slog.Logger
}

// Switch выполняет переключение на бэкап-хост (CNAME)
func (f *Failover) Switch(ctx context.Context, host model.Host) error {
	backup, err := f.selectBackupHost(host)
	if err != nil {
		return err
	}

	record := model.Record{
		Name:   host.DNS.Record,
		Type:   "CNAME",
		Target: backup.DNS.Record,
		TTL:    300,
	}

	// 1. Обновляем DNS
	if err := f.Provider.UpdateRecords(ctx, host.DNS.Zone, []model.Record{record}); err != nil {
		return err
	}

	// 2. Отправляем уведомление (ошибки не фатальны)
	if err := f.Notifier.SendFailover(ctx, host, *backup); err != nil {
		if f.Log != nil {
			f.Log.Error("failed to send failover notification", "host", host.Name, "error", err)
		}
		// Не возвращаем ошибку, т.к. DNS уже переключен
	}

	return nil
}

// Restore восстанавливает A-запись с оригинальным IP
func (f *Failover) Restore(ctx context.Context, host model.Host) error {
	record := model.Record{
		Name: host.DNS.Record,
		Type: "A",
		IP:   host.DNS.IP,
		TTL:  300,
	}

	// 1. Обновляем DNS
	if err := f.Provider.UpdateRecords(ctx, host.DNS.Zone, []model.Record{record}); err != nil {
		return err
	}

	// 2. Отправляем уведомление (ошибки не фатальны)
	if err := f.Notifier.SendRecovery(ctx, host); err != nil {
		if f.Log != nil {
			f.Log.Error("failed to send recovery notification", "host", host.Name, "error", err)
		}
		// Не возвращаем ошибку
	}

	return nil
}

// selectBackupHost выбирает следующий хост по порядку (циклически)
func (f *Failover) selectBackupHost(current model.Host) (*model.Host, error) {
	if len(f.Hosts) < 2 {
		return nil, fmt.Errorf("not enough hosts for failover")
	}
	for i, h := range f.Hosts {
		if h.Name == current.Name {
			nextIdx := (i + 1) % len(f.Hosts)
			return &f.Hosts[nextIdx], nil
		}
	}
	return nil, fmt.Errorf("current host %s not found in hosts list", current.Name)
}