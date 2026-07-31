package dns

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
	"io"

	"dns-failover/internal/model"
)

type SelectelProvider struct {

	DomainURL string
	TokenURL string

	AccountID string
	ProjectName string

	Username string
	Password string

	Client *http.Client
}

func NewSelectel(
	accountID string,
	projectName string,
	username string,
	password string,
) *SelectelProvider {

    return &SelectelProvider{

       	DomainURL: "https://api.selectel.ru/domains/v2",

        TokenURL: "https://cloud.api.selcloud.ru/identity/v3/auth/tokens",

        AccountID: accountID,
		ProjectName: projectName,
        Username: username,

        Password: password,

        Client: &http.Client{
            Timeout: 10 * time.Second,
        },
    }
}



func (s *SelectelProvider) getToken(
	ctx context.Context,
) (string, error) {

	body := map[string]interface{}{
		"auth": map[string]interface{}{
			"identity": map[string]interface{}{
				"methods": []string{
					"password",
				},

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

	data, err := json.Marshal(body)

	if err != nil {
		return "", err
	}


	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		s.TokenURL,
		bytes.NewBuffer(data),
	)

	if err != nil {
		return "", err
	}


	req.Header.Set(
		"Content-Type",
		"application/json",
	)


	resp, err := s.Client.Do(req)

	if err != nil {
		return "", err
	}

	defer resp.Body.Close()




	if resp.StatusCode != http.StatusCreated {

		body, _ := io.ReadAll(resp.Body)

		return "",
			fmt.Errorf(
				"auth failed: %s response=%s",
				resp.Status,
				string(body),
			)
	}


	token := resp.Header.Get(
		"X-Subject-Token",
	)


	if token == "" {
		return "",
			fmt.Errorf(
				"empty token received",
			)
	}


	return token, nil
}





type rrsetResponse struct {
    Count      int `json:"count"`
    NextOffset int `json:"next_offset"`

    Result []struct {
        ID   string `json:"id"`
        Name string `json:"name"`
        TTL  int    `json:"ttl"`
        Type string `json:"type"`

        Records []struct {
            Content  string `json:"content"`
            Disabled bool   `json:"disabled"`
        } `json:"records"`
    } `json:"result"`
}





type selectelRecord struct {

	Content string `json:"content"`

	Disabled bool `json:"disabled"`
}






// GetRecords получает все A записи
func (s *SelectelProvider) GetRecords(
	ctx context.Context,
	zone string,
	record string,
) ([]model.Record, error) {

	token, err := s.getToken(ctx)

	if err != nil {
		return nil, err
	}



	url := fmt.Sprintf(
		"%s/zones/%s/rrset",
		s.DomainURL,
		zone,
	)

	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodGet,
		url,
		nil,
	)

	if err != nil {
		return nil, err
	}



	req.Header.Set(
		"X-Auth-Token",
		token,
	)



	resp, err := s.Client.Do(req)

	if err != nil {
		return nil, err
	}


	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	resp.Body = io.NopCloser(bytes.NewBuffer(body))

	var result rrsetResponse


	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {

		return nil, err
	}




	for _, rr := range result.Result {


		if rr.Name != record || rr.Type != "A" {
			continue
		}
		records := make([]model.Record, 0, len(rr.Records))

		for _, r := range rr.Records {
			records = append(records, model.Record{
					ID: rr.ID,
					TTL: rr.TTL,
					IP: r.Content,
					Disabled: r.Disabled,
				},
			)
		}


		return records, nil
	}



	return nil,
		fmt.Errorf(
			"record not found",
		)
}







// UpdateRecords меняет только disabled
func (s *SelectelProvider) UpdateRecords(
	ctx context.Context,
	zone string,
	records []model.Record,
) error {
	if len(records) == 0 {
		return fmt.Errorf("no records")
	}

	rrsetID := records[0].ID
	token, err := s.getToken(ctx)

	if err != nil {
		return err
	}



	selectelRecords := make(
		[]selectelRecord,
		0,
	)



	for _, r := range records {


		selectelRecords = append(
			selectelRecords,
			selectelRecord{

				Content: r.IP,

				Disabled: r.Disabled,
			},
		)
	}




	body := map[string]interface{}{
		"ttl": records[0].TTL,
		"records": selectelRecords,
	}



	debug, _ := json.MarshalIndent(body, "", "  ")
	fmt.Println(string(debug))
	data, err := json.Marshal(body)

	if err != nil {
		return err
	}




	url := fmt.Sprintf(
		"%s/zones/%s/rrset/%s",
		s.DomainURL,
		zone,
		rrsetID,
	)




	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodPatch,
		url,
		bytes.NewBuffer(data),
	)

	if err != nil {
		return err
	}




	req.Header.Set(
		"Content-Type",
		"application/json",
	)


	req.Header.Set(
		"X-Auth-Token",
		token,
	)




	resp, err := s.Client.Do(req)

	if err != nil {
		return err
	}


	defer resp.Body.Close()

	bodyResp, _ := io.ReadAll(resp.Body)


	if resp.StatusCode >= 300 {


		return fmt.Errorf(
			"dns update failed: %s",
			resp.Status,
		)
	}

	if resp.StatusCode != http.StatusNoContent {
		return fmt.Errorf(
			"dns update failed: %s body=%s",
			resp.Status,
			string(bodyResp),
		)
	}


	return nil
}