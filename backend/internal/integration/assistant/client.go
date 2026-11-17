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
)

type Client struct {
	endpoint string
	http     *http.Client
}

type Request struct {
	Skill   string `json:"skill,omitempty"`
	Message string `json:"message"`
}

type Source struct {
	ID    string `json:"id"`
	Title string `json:"title"`
}

type Answer struct {
	Skill   string   `json:"skill"`
	Mode    string   `json:"mode"`
	Answer  string   `json:"answer"`
	Sources []Source `json:"sources"`
}

var ErrUnknownSkill = errors.New("skill belum tersedia")

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

func (c *Client) Chat(ctx context.Context, input Request) (Answer, error) {
	body, err := json.Marshal(input)
	if err != nil {
		return Answer{}, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.endpoint, bytes.NewReader(body))
	if err != nil {
		return Answer{}, err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.http.Do(req)
	if err != nil {
		return Answer{}, err
	}
	defer resp.Body.Close()

	var payload struct {
		Data  Answer `json:"data"`
		Error struct {
			Code string `json:"code"`
		} `json:"error"`
	}
	if err := json.NewDecoder(io.LimitReader(resp.Body, 64<<10)).Decode(&payload); err != nil {
		return Answer{}, err
	}
	if resp.StatusCode == http.StatusBadRequest && payload.Error.Code == "UNKNOWN_SKILL" {
		return Answer{}, ErrUnknownSkill
	}
	if resp.StatusCode != http.StatusOK || payload.Data.Skill == "" || payload.Data.Answer == "" {
		return Answer{}, errors.New("respons AI Service tidak valid")
	}
	return payload.Data, nil
}
