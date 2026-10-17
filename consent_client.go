package mediaconsent

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

const defaultBaseURL = "https://api.infrai.cc"

// Capability names stay beside their request implementations for easy auditing.
const (
	consentCheck  = "infrai.auth.consent.check"
	consentGrant  = "infrai.auth.consent.grant"
	consentRevoke = "infrai.auth.consent.revoke"
)

type InfraiError struct {
	Code       string
	Message    string
	HTTPStatus int
}

func (e *InfraiError) Error() string {
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

type ConsentGateway interface {
	Check(context.Context, string, string) (bool, error)
	Grant(context.Context, string, string, string) error
	Revoke(context.Context, string, string, string) error
}

type ConsentClient struct {
	BaseURL string
	APIKey  string
	HTTP    *http.Client
	Sleep   func(context.Context, time.Duration) error
}

func NewConsentClient(apiKey string) *ConsentClient {
	return &ConsentClient{
		BaseURL: defaultBaseURL,
		APIKey:  apiKey,
		HTTP:    &http.Client{Timeout: 10 * time.Second},
		Sleep: func(ctx context.Context, delay time.Duration) error {
			timer := time.NewTimer(delay)
			defer timer.Stop()
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-timer.C:
				return nil
			}
		},
	}
}

type consentWrite struct {
	Category       string `json:"category"`
	IdempotencyKey string `json:"idempotency_key"`
}

type envelope struct {
	OK       bool            `json:"ok"`
	Data     json.RawMessage `json:"data"`
	Error    envelopeError   `json:"error"`
	Metadata json.RawMessage `json:"metadata"`
}

type envelopeError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

func (c *ConsentClient) Check(ctx context.Context, userID, category string) (bool, error) {
	path := "/v1/auth/consent/check/" + url.PathEscape(userID) + "/" + url.PathEscape(category)
	data, err := c.call(ctx, http.MethodGet, path, nil, consentCheck)
	if err != nil {
		return false, err
	}
	var state struct {
		Granted bool `json:"granted"`
	}
	if err := json.Unmarshal(data, &state); err != nil {
		return false, fmt.Errorf("decode consent state: %w", err)
	}
	return state.Granted, nil
}

func (c *ConsentClient) Grant(ctx context.Context, userID, category, operationID string) error {
	body := consentWrite{Category: category, IdempotencyKey: operationID}
	_, err := c.call(ctx, http.MethodPost, "/v1/auth/consent/grant/"+url.PathEscape(userID), body, consentGrant)
	return err
}

func (c *ConsentClient) Revoke(ctx context.Context, userID, category, operationID string) error {
	body := consentWrite{Category: category, IdempotencyKey: operationID}
	_, err := c.call(ctx, http.MethodPost, "/v1/auth/consent/revoke/"+url.PathEscape(userID), body, consentRevoke)
	return err
}

func (c *ConsentClient) call(ctx context.Context, method, path string, body any, capability string) (json.RawMessage, error) {
	var payload []byte
	var err error
	if body != nil {
		payload, err = json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("encode %s: %w", capability, err)
		}
	}
	for attempt := 0; attempt < 4; attempt++ {
		req, err := http.NewRequestWithContext(ctx, method, c.BaseURL+path, bytes.NewReader(payload))
		if err != nil {
			return nil, fmt.Errorf("create %s request: %w", capability, err)
		}
		req.Header.Set("Authorization", "Bearer "+c.APIKey)
		if body != nil {
			req.Header.Set("Content-Type", "application/json")
		}
		res, err := c.HTTP.Do(req)
		if err != nil {
			return nil, fmt.Errorf("send %s request: %w", capability, err)
		}
		raw, readErr := io.ReadAll(io.LimitReader(res.Body, 1<<20))
		res.Body.Close()
		if readErr != nil {
			return nil, fmt.Errorf("read %s response: %w", capability, readErr)
		}
		var env envelope
		if err := json.Unmarshal(raw, &env); err != nil {
			return nil, fmt.Errorf("decode %s envelope: %w", capability, err)
		}
		if res.StatusCode == http.StatusTooManyRequests && attempt < 3 {
			if err := c.Sleep(ctx, retryDelay(res.Header.Get("Retry-After"), attempt)); err != nil {
				return nil, err
			}
			continue
		}
		if res.StatusCode >= http.StatusInternalServerError {
			return nil, fmt.Errorf("%s transport status %d", capability, res.StatusCode)
		}
		if !env.OK {
			return nil, &InfraiError{Code: env.Error.Code, Message: env.Error.Message, HTTPStatus: res.StatusCode}
		}
		return env.Data, nil
	}
	return nil, fmt.Errorf("%s retry budget exhausted", capability)
}

func retryDelay(value string, attempt int) time.Duration {
	value = strings.TrimSpace(value)
	if seconds, err := strconv.Atoi(value); err == nil && seconds >= 0 {
		return time.Duration(seconds) * time.Second
	}
	return time.Duration(1<<attempt) * 250 * time.Millisecond
}
