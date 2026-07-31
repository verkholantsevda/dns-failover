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
)

type Telegram struct {
	Token    string
	ChatID   int64
	ImageURL string
	Support  Support
	Client   *http.Client
}

func NewTelegram(
	token string,
	chatID int64,
	imageURL string,
	support Support,
) *Telegram {

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

func (t *Telegram) sendImage(
	ctx context.Context,
	imagePath string,
	caption string,
) error {

	file, err := os.Open(imagePath)
	if err != nil {
		return err
	}
	defer file.Close()

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	if err := writer.WriteField(
		"chat_id",
		fmt.Sprintf("%d", t.ChatID),
	); err != nil {
		return err
	}

	if err := writer.WriteField(
		"caption",
		caption,
	); err != nil {
		return err
	}

	if err := writer.WriteField(
		"parse_mode",
		"HTML",
	); err != nil {
		return err
	}

	part, err := writer.CreateFormFile(
		"photo",
		filepath.Base(imagePath),
	)

	if err != nil {
		return err
	}

	_, err = io.Copy(
		part,
		file,
	)

	if err != nil {
		return err
	}

	err = writer.Close()

	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		fmt.Sprintf(
			"https://api.telegram.org/bot%s/sendPhoto",
			t.Token,
		),
		body,
	)

	if err != nil {
		return err
	}

	req.Header.Set(
		"Content-Type",
		writer.FormDataContentType(),
	)

	fmt.Println("Content-Length:", body.Len())
	fmt.Println("Image:", imagePath)
	resp, err := t.Client.Do(req)

	if err != nil {
		return fmt.Errorf(
			"telegram sendPhoto request failed: %w",
			err,
		)
	}

	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)

	if resp.StatusCode >= 300 {

		return fmt.Errorf(
			"telegram sendPhoto failed: %s body=%s",
			resp.Status,
			string(respBody),
		)
	}

	return nil
}

// Базовая отправка Telegram
func (t *Telegram) Send(
	ctx context.Context,
	message string,
) error {

	body := map[string]any{
		"chat_id":    t.ChatID,
		"text":       message,
		"parse_mode": "HTML",
	}

	data, err := json.Marshal(body)

	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		fmt.Sprintf(
			"https://api.telegram.org/bot%s/sendMessage",
			t.Token,
		),
		bytes.NewBuffer(data),
	)

	if err != nil {
		return err
	}

	req.Header.Set(
		"Content-Type",
		"application/json",
	)

	resp, err := t.Client.Do(req)

	if err != nil {
		return err
	}

	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)

	if resp.StatusCode >= 300 {

		return fmt.Errorf(
			"telegram error: %s body=%s",
			resp.Status,
			string(respBody),
		)
	}

	return nil
}

// Уведомление о переключении
func (t *Telegram) SendFailover(
	ctx context.Context,
	fromCountry string,
	toCountry string,
) error {

	message := FailoverMessage(
		fromCountry,
		toCountry,
		t.Support,
	)

	return t.sendImage(
		ctx,
		"internal/notifier/images/failover.jpg",
		message,
	)
}

// Уведомление о восстановлении
func (t *Telegram) SendRecovery(
	ctx context.Context,
	fromCountry string,
	toCountry string,
) error {

	message := RecoveryMessage(
		fromCountry,
		toCountry,
		t.Support,
	)

	return t.sendImage(
		ctx,
		"internal/notifier/images/recovery.jpg",
		message,
	)
}
