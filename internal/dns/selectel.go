package dns

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"dns-failover/internal/model"
)

type SelectelProvider struct {
	DomainURL   string
	TokenURL    string
	AccountID   string
	ProjectName string
	Username    string
	Password    string
	Client      *http.Client
}

func NewSelectel(accountID, projectName, username, password string) *SelectelProvider {
	return &SelectelProvider{
		DomainURL:   "https://api.selectel.ru/domains/v2",
		TokenURL:    "https://cloud.api.selcloud.ru/identity/v3/auth/tokens",
		AccountID:   accountID,
		ProjectName: projectName,
		Username:    username,
		Password:    password,
		Client: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// getToken получает токен авторизации
func (s *SelectelProvider) getToken(ctx context.Context) (string, error) {
	body := map[string]interface{}{
		"auth": map[string]interface{}{
			"identity": map[string]interface{}{
				"methods": []string{"password"},
				"password": map[string]interface{}{
					"user": map[string]interface{}{
						"name": s.Username,
						"domain": map[string]string{
							"name": s.AccountID,
						},
						"password": s.Password,
					},
				},
			},
			"scope": map[string]interface{}{
				"project": map[string]interface{}{
					"name": s.ProjectName,
					"domain": map[string]string{
						"name": s.AccountID,
					},
				},
			},
		},
	}
	data, _ := json.Marshal(body)
	req, _ := http.NewRequestWithContext(ctx, http.MethodPost, s.TokenURL, bytes.NewBuffer(data))
	req.Header.Set("Content-Type", "application/json")
	resp, err := s.Client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("auth failed: %s response=%s", resp.Status, string(body))
	}
	token := resp.Header.Get("X-Subject-Token")
	if token == "" {
		return "", fmt.Errorf("empty token received")
	}
	return token, nil
}

// rrsetResponse используется для получения списка RRset
type rrsetResponse struct {
	Count      int `json:"count"`
	NextOffset int `json:"next_offset"`
	Result     []struct {
		ID      string `json:"id"`
		Name    string `json:"name"`
		TTL     int    `json:"ttl"`
		Type    string `json:"type"`
		Records []struct {
			Content  string `json:"content"`
			Disabled bool   `json:"disabled"`
		} `json:"records"`
	} `json:"result"`
}

// GetRecords ищет запись по имени и возвращает одну запись (если найдена).
// Если записи нет, возвращает пустой слайс и nil.
func (s *SelectelProvider) GetRecords(ctx context.Context, zone, record string) ([]model.Record, error) {
	token, err := s.getToken(ctx)
	if err != nil {
		return nil, err
	}

	url := fmt.Sprintf("%s/zones/%s/rrset", s.DomainURL, zone)
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	req.Header.Set("X-Auth-Token", token)
	resp, err := s.Client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result rrsetResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	for _, rr := range result.Result {
		if rr.Name == record {
			if len(rr.Records) == 0 {
				continue
			}
			rec := model.Record{
				ID:   rr.ID,
				Name: rr.Name,
				Type: rr.Type,
				TTL:  rr.TTL,
			}
			if rr.Type == "A" {
				rec.IP = rr.Records[0].Content
			} else if rr.Type == "CNAME" {
				rec.Target = rr.Records[0].Content
			}
			return []model.Record{rec}, nil
		}
	}
	return []model.Record{}, nil
}

// UpdateRecords обновляет запись: удаляет старую (если есть) и создаёт новую.
func (s *SelectelProvider) UpdateRecords(ctx context.Context, zone string, records []model.Record) error {
	if len(records) != 1 {
		return fmt.Errorf("expected exactly one record, got %d", len(records))
	}
	newRec := records[0]
	if newRec.Name == "" {
		return fmt.Errorf("record name is empty")
	}

	token, err := s.getToken(ctx)
	if err != nil {
		return err
	}

	// 1. Получить текущую запись с таким именем
	existing, err := s.GetRecords(ctx, zone, newRec.Name)
	if err != nil {
		return err
	}

	// 2. Если существует, удалить её через DELETE
	if len(existing) > 0 {
		oldRec := existing[0]
		url := fmt.Sprintf("%s/zones/%s/rrset/%s", s.DomainURL, zone, oldRec.ID)
		req, _ := http.NewRequestWithContext(ctx, http.MethodDelete, url, nil)
		req.Header.Set("X-Auth-Token", token)
		resp, err := s.Client.Do(req)
		if err != nil {
			return err
		}
		resp.Body.Close()
		// DELETE возвращает 204 No Content или 200 OK при успехе
		if resp.StatusCode != http.StatusNoContent && resp.StatusCode != http.StatusOK {
			return fmt.Errorf("failed to delete old record: %s", resp.Status)
		}
	}

	// 3. Создать новую запись (POST)
	rrset := map[string]interface{}{
		"name": newRec.Name,
		"type": newRec.Type,
		"ttl":  newRec.TTL,
		"records": []map[string]interface{}{
			{
				"content":  "",
				"disabled": false,
			},
		},
	}
	if newRec.Type == "A" {
		rrset["records"].([]map[string]interface{})[0]["content"] = newRec.IP
	} else if newRec.Type == "CNAME" {
		rrset["records"].([]map[string]interface{})[0]["content"] = newRec.Target
	} else {
		return fmt.Errorf("unsupported record type: %s", newRec.Type)
	}

	data, _ := json.Marshal(rrset)
	url := fmt.Sprintf("%s/zones/%s/rrset", s.DomainURL, zone)
	req, _ := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewBuffer(data))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Auth-Token", token)
	resp, err := s.Client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to create record: %s body=%s", resp.Status, string(body))
	}
	return nil
}