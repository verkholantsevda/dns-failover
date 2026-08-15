package notifier

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"dns-failover/internal/model"
)

type Telegram struct {
	Token    string
	ChatID   int64
	ImageURL string
	Support  Support
	Client   *http.Client
}

func NewTelegram(token string, chatID int64, imageURL string, support Support) *Telegram {
	return &Telegram{
		Token:    token,
		ChatID:   chatID,
		ImageURL: imageURL,
		Support:  support,
		Client: &http.Client{
			Timeout: 15 * time.Second,
			Transport: &http.Transport{
				ForceAttemptHTTP2: false,
			},
		},
	}
}

func (t *Telegram) sendImage(ctx context.Context, imagePath string, caption string) error {
	file, err := os.Open(imagePath)
	if err != nil {
		return err
	}
	defer file.Close()

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	if err := writer.WriteField("chat_id", fmt.Sprintf("%d", t.ChatID)); err != nil {
		return err
	}
	if err := writer.WriteField("caption", caption); err != nil {
		return err
	}
	if err := writer.WriteField("parse_mode", "HTML"); err != nil {
		return err
	}

	part, err := writer.CreateFormFile("photo", filepath.Base(imagePath))
	if err != nil {
		return err
	}
	if _, err := io.Copy(part, file); err != nil {
		return err
	}
	if err := writer.Close(); err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		fmt.Sprintf("https://api.telegram.org/bot%s/sendPhoto", t.Token), body)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())

	fmt.Printf("[TELEGRAM] Sending image: %s, caption length: %d\n", imagePath, len(caption))

	resp, err := t.Client.Do(req)
	if err != nil {
		return fmt.Errorf("telegram sendPhoto request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)

	if resp.StatusCode >= 300 {
		return fmt.Errorf("telegram sendPhoto failed: %s body=%s", resp.Status, string(respBody))
	}

	// Логируем успех
	fmt.Printf("[TELEGRAM] Image sent successfully: %s\n", imagePath)
	return nil
}

func (t *Telegram) Send(ctx context.Context, message string) error {
	body := map[string]any{
		"chat_id":    t.ChatID,
		"text":       message,
		"parse_mode": "HTML",
	}
	data, _ := json.Marshal(body)
	req, _ := http.NewRequestWithContext(ctx, http.MethodPost,
		fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", t.Token), bytes.NewBuffer(data))
	req.Header.Set("Content-Type", "application/json")

	resp, err := t.Client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	respBody, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 300 {
		return fmt.Errorf("telegram error: %s body=%s", resp.Status, string(respBody))
	}
	return nil
}

// Уведомление о переключении
func (t *Telegram) SendFailover(ctx context.Context, host, backup model.Host) error {
	fmt.Printf("[TELEGRAM] SendFailover called for host %s\n", host.Name)
	message := FailoverMessage(host.Country, backup.Country, t.Support)
	return t.sendImage(ctx, "internal/notifier/images/failover.jpg", message)
}

func (t *Telegram) SendRecovery(ctx context.Context, host model.Host) error {
	fmt.Printf("[TELEGRAM] SendRecovery called for host %s\n", host.Name)
	message := RecoveryMessage(host.Country, t.Support)
	return t.sendImage(ctx, "internal/notifier/images/recovery.jpg", message)
}

// ICMP-уведомления через Telegram не отправляем (заглушки)
func (t *Telegram) SendICMPDown(ctx context.Context, host model.Host) error { return nil }
func (t *Telegram) SendICMPUp(ctx context.Context, host model.Host) error   { return nil }