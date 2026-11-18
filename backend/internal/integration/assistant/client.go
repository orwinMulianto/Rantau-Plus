package assistant

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/url"
	"time"

	"rantauplus/backend/internal/modules/assistant"
)

type Client struct {
	endpoint string
	http     *http.Client
}

func NewClient(baseURL string) (*Client, error) {
	base, err := url.Parse(baseURL)
	if err != nil || (base.Scheme != "http" && base.Scheme != "https") ||
		base.Hostname() == "" || base.User != nil || base.RawQuery != "" || base.Fragment != "" {
		return nil, errors.New("AI_SERVICE_URL harus berupa URL HTTP atau HTTPS yang valid")
	}
	return &Client{
		endpoint: base.JoinPath("internal", "v1", "chat").String(),
		http: &http.Client{
			Timeout: 8 * time.Second,
			CheckRedirect: func(*http.Request, []*http.Request) error {
				return http.ErrUseLastResponse
			},
		},
	}, nil
}

func (c *Client) Chat(ctx context.Context, input assistant.Request) (assistant.Answer, error) {
	body, err := json.Marshal(input)
	if err != nil {
		return assistant.Answer{}, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.endpoint, bytes.NewReader(body))
	if err != nil {
		return assistant.Answer{}, err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.http.Do(req)
	if err != nil {
		return assistant.Answer{}, err
	}
	defer resp.Body.Close()

	var payload struct {
		Data  assistant.Answer `json:"data"`
		Error struct {
			Code string `json:"code"`
		} `json:"error"`
	}
	if err := json.NewDecoder(io.LimitReader(resp.Body, 64<<10)).Decode(&payload); err != nil {
		return assistant.Answer{}, err
	}
	if resp.StatusCode == http.StatusBadRequest && payload.Error.Code == "UNKNOWN_SKILL" {
		return assistant.Answer{}, assistant.ErrUnknownSkill
	}
	if resp.StatusCode != http.StatusOK || payload.Data.Skill == "" || payload.Data.Answer == "" {
		return assistant.Answer{}, errors.New("respons AI Service tidak valid")
	}
	return payload.Data, nil
}
