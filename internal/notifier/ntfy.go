package notifier

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"time"

	"dns-failover/internal/model"
)

type Ntfy struct {
	url   string
	topic string
	token string
}

func New(url, topic, token string) *Ntfy {
	return &Ntfy{
		url:   url,
		topic: topic,
		token: token,
	}
}

func (n *Ntfy) send(title, message string) error {
	req, err := http.NewRequest("POST", fmt.Sprintf("%s/%s", n.url, n.topic), bytes.NewBufferString(message))
	if err != nil {
		return err
	}
	if n.token != "" {
		req.Header.Set("Authorization", "Bearer "+n.token)
	}
	req.Header.Set("Title", title)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		return fmt.Errorf("ntfy status: %s", resp.Status)
	}
	return nil
}

// DNS переключение
func (n *Ntfy) SendFailover(ctx context.Context, host, backup model.Host) error {
	title := "Server DOWN"
	fromFlag, _ := countryInfo(host.Country)
	toFlag, toName := countryInfo(backup.Country)
	message := fmt.Sprintf(
		"🔴 %s %s (%s) is unreachable\nDNS Switched to %s %s\n%s",
		fromFlag, host.Name, host.DNS.IP,
		toFlag, toName,
		time.Now().Format("15:04"),
	)
	return n.send(title, message)
}

// DNS восстановление
func (n *Ntfy) SendRecovery(ctx context.Context, host model.Host) error {
	title := "Server UP"
	flag, _ := countryInfo(host.Country)
	message := fmt.Sprintf(
		"🟢 %s %s (%s) is back online\nDNS Restored\n%s",
		flag, host.Name, host.DNS.IP,
		time.Now().Format("15:04"),
	)
	return n.send(title, message)
}

// ICMP DOWN
func (n *Ntfy) SendICMPDown(ctx context.Context, host model.Host) error {
	title := "Server DOWN"
	flag, _ := countryInfo(host.Country)
	message := fmt.Sprintf(
		"🔴 %s %s (%s) is unreachable\n%s",
		flag, host.Name, host.DNS.IP,
		time.Now().Format("15:04"),
	)
	return n.send(title, message)
}

// ICMP UP
func (n *Ntfy) SendICMPUp(ctx context.Context, host model.Host) error {
	title := "Server UP"
	flag, _ := countryInfo(host.Country)
	message := fmt.Sprintf(
		"🟢 %s %s (%s) is back online\n%s",
		flag, host.Name, host.DNS.IP,
		time.Now().Format("15:04"),
	)
	return n.send(title, message)
}

// Вспомогательная функция – использует существующий словарь Countries из flags.go
func countryInfo(code string) (flag, name string) {
	c, ok := Countries[code]
	if !ok {
		return "🏳️", code
	}
	return c.Flag, c.Name
}