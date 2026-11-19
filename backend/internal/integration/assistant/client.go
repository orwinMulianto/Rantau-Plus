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
	endpoint          string
	companionEndpoint string
	http              *http.Client
}

func NewClient(baseURL string) (*Client, error) {
	base, err := url.Parse(baseURL)
	if err != nil || (base.Scheme != "http" && base.Scheme != "https") ||
		base.Hostname() == "" || base.User != nil || base.RawQuery != "" || base.Fragment != "" {
		return nil, errors.New("AI_SERVICE_URL harus berupa URL HTTP atau HTTPS yang valid")
	}
	return &Client{
		endpoint:          base.JoinPath("internal", "v1", "chat").String(),
		companionEndpoint: base.JoinPath("internal", "v1", "companion").String(),
		http: &http.Client{
			Timeout: 65 * time.Second,
			CheckRedirect: func(*http.Request, []*http.Request) error {
				return http.ErrUseLastResponse
			},
		},
	}, nil
}

func (c *Client) Chat(ctx context.Context, input assistant.Request) (assistant.Answer, error) {
	var answer assistant.Answer
	err := c.post(ctx, c.endpoint, input, &answer)
	if err == nil && (answer.Skill == "" || answer.Answer == "") {
		err = errors.New("jawaban AI Service kosong")
	}
	return answer, err
}

func (c *Client) Plan(ctx context.Context, input assistant.PlanRequest) (assistant.Plan, error) {
	var plan assistant.Plan
	err := c.post(ctx, c.companionEndpoint, input, &plan)
	return plan, err
}

func (c *Client) post(ctx context.Context, endpoint string, input, output any) error {
	body, err := json.Marshal(input)
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.http.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	var payload struct {
		Data  json.RawMessage `json:"data"`
		Error struct {
			Code string `json:"code"`
		} `json:"error"`
	}
	if err := json.NewDecoder(io.LimitReader(resp.Body, 64<<10)).Decode(&payload); err != nil {
		return err
	}
	if resp.StatusCode == http.StatusBadRequest && payload.Error.Code == "UNKNOWN_SKILL" {
		return assistant.ErrUnknownSkill
	}
	if resp.StatusCode != http.StatusOK {
		return errors.New("AI Service gagal memproses permintaan")
	}
	return json.Unmarshal(payload.Data, output)
}
