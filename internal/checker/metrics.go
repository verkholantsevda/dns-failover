package checker

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"time"
)

type promResponse struct {
	Status string `json:"status"`

	Data struct {
		ResultType string `json:"resultType"`

		Result []struct {
			Metric map[string]string `json:"metric"`
			Value  []interface{}     `json:"value"`
		} `json:"result"`
	} `json:"data"`
}

type PrometheusChecker struct {
	URL    string
	Client *http.Client
}

func NewPrometheus(url string) *PrometheusChecker {

	return &PrometheusChecker{
		URL: url,
		Client: &http.Client{
			Timeout: 5 * time.Second,
		},
	}

}

func (p *PrometheusChecker) Check(
	ctx context.Context,
	query string,
) (bool, error) {

	values := url.Values{}
	values.Set("query", query)
	fmt.Println("PromQL:", query)
	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodGet,
		p.URL+"/api/v1/query?"+values.Encode(),
		nil,
	)
	if err != nil {
		return false, err
	}

	resp, err := p.Client.Do(req)
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return false, fmt.Errorf("unexpected status: %d", resp.StatusCode)
	}

	var result promResponse

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return false, err
	}

	if len(result.Data.Result) == 0 {
		return false, nil
	}

	value, ok := result.Data.Result[0].Value[1].(string)
	if !ok {
		return false, fmt.Errorf("invalid prometheus response")
	}

	return value == "1", nil
}