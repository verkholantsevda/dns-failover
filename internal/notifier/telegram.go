package notifier

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
	"io"
)

type Telegram struct {
	Token  string
	ChatID int64
	Client *http.Client
}

func NewTelegram(token string, chatID int64) *Telegram {
	return &Telegram{
		Token:  token,
		ChatID: chatID,
		Client: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

func (t *Telegram) Send(ctx context.Context, message string) error {
	body := map[string]any{
		"chat_id":    t.ChatID,
		"text":       message,
		"parse_mode": "HTML",
	}

	data, _ := json.Marshal(body)

	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", t.Token),
		bytes.NewBuffer(data),
	)
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := t.Client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)

	fmt.Println("TELEGRAM STATUS:", resp.Status)
	fmt.Println("TELEGRAM BODY:", string(respBody))

	if resp.StatusCode >= 300 {
		return fmt.Errorf(
			"telegram error: %s body=%s",
			resp.Status,
			string(respBody),
		)
	}

	return nil
}