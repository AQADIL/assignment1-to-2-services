package paymentclient

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"order-service/internal/domain"
)

type Client struct {
	baseURL string
	httpCli *http.Client
}

func New(baseURL string) *Client {
	baseURL = strings.TrimRight(baseURL, "/")
	return &Client{
		baseURL: baseURL,
		httpCli: &http.Client{Timeout: 2 * time.Second},
	}
}

func (c *Client) CreatePayment(ctx context.Context, req domain.PaymentCreateRequest) (domain.PaymentCreateResponse, error) {
	url := c.baseURL + "/payments"

	payload, err := json.Marshal(req)
	if err != nil {
		return domain.PaymentCreateResponse{}, err
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(payload))
	if err != nil {
		return domain.PaymentCreateResponse{}, err
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := c.httpCli.Do(httpReq)
	if err != nil {
		return domain.PaymentCreateResponse{}, err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return domain.PaymentCreateResponse{}, err
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		if len(respBody) == 0 {
			return domain.PaymentCreateResponse{}, fmt.Errorf("payment service returned status %d", resp.StatusCode)
		}
		return domain.PaymentCreateResponse{}, errors.New(string(respBody))
	}

	var out domain.PaymentCreateResponse
	if err := json.Unmarshal(respBody, &out); err != nil {
		return domain.PaymentCreateResponse{}, err
	}
	return out, nil
}
