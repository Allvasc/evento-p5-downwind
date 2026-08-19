package asaas

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"time"
)

type HTTPClient struct {
	baseURL string
	apiKey  string
	http    *http.Client
	log     *slog.Logger
}

func NewHTTPClient(baseURL, apiKey string, log *slog.Logger) *HTTPClient {
	return &HTTPClient{
		baseURL: baseURL,
		apiKey:  apiKey,
		http:    &http.Client{Timeout: 15 * time.Second},
		log:     log,
	}
}

func (c *HTTPClient) do(ctx context.Context, method, path string, body any, out any) error {
	var reader io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return err
		}
		reader = bytes.NewReader(b)
	}

	req, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, reader)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("access_token", c.apiKey)

	res, err := c.http.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()

	respBody, err := io.ReadAll(res.Body)
	if err != nil {
		return err
	}
	if res.StatusCode >= 300 {
		return fmt.Errorf("asaas api error (status %d): %s", res.StatusCode, string(respBody))
	}
	if out != nil {
		return json.Unmarshal(respBody, out)
	}
	return nil
}

func (c *HTTPClient) CreateCustomer(ctx context.Context, p CustomerParams) (string, error) {
	var out struct {
		ID string `json:"id"`
	}
	payload := map[string]string{
		"name":        p.Name,
		"email":       p.Email,
		"mobilePhone": p.Phone,
	}
	if p.CPF != "" {
		payload["cpfCnpj"] = p.CPF
	}
	if err := c.do(ctx, http.MethodPost, "/customers", payload, &out); err != nil {
		return "", err
	}
	return out.ID, nil
}

func (c *HTTPClient) CreateCharge(ctx context.Context, p ChargeParams) (Charge, error) {
	var created struct {
		ID     string `json:"id"`
		Status string `json:"status"`
	}
	payload := map[string]any{
		"customer":          p.CustomerID,
		"billingType":       p.BillingType,
		"value":             float64(p.ValueCents) / 100,
		"description":       p.Description,
		"externalReference": p.ExternalRef,
		"dueDate":           time.Now().Format("2006-01-02"),
	}
	if err := c.do(ctx, http.MethodPost, "/payments", payload, &created); err != nil {
		return Charge{}, err
	}

	charge := Charge{PaymentID: created.ID, Status: created.Status}

	if p.BillingType == "PIX" {
		// A Asaas às vezes leva um instante para terminar de gerar o QR code logo após
		// criar a cobrança — a primeira tentativa pode chegar cedo demais, então tenta
		// de novo algumas vezes antes de desistir (e loga, em vez de engolir o erro).
		var pix struct {
			EncodedImage string `json:"encodedImage"`
			Payload      string `json:"payload"`
		}
		var pixErr error
		for attempt := 1; attempt <= 4; attempt++ {
			pixErr = c.do(ctx, http.MethodGet, "/payments/"+created.ID+"/pixQrCode", nil, &pix)
			if pixErr == nil {
				break
			}
			time.Sleep(time.Duration(attempt) * 400 * time.Millisecond)
		}
		if pixErr != nil {
			c.log.Error("asaas fetch pix qr code failed after retries", "error", pixErr, "paymentId", created.ID)
		} else {
			charge.PixQRImage = pix.EncodedImage
			charge.PixCopyPaste = pix.Payload
		}
	}

	return charge, nil
}

// RefundPayment reverses a confirmed Pix payment. Asaas returns the refund status
// asynchronously via webhook (PAYMENT_REFUNDED), but the request itself is what
// triggers the money movement — the caller flips local order/entitlement state right
// after this succeeds rather than waiting for that webhook round-trip.
func (c *HTTPClient) RefundPayment(ctx context.Context, paymentID string) error {
	return c.do(ctx, http.MethodPost, "/payments/"+paymentID+"/refund", map[string]string{}, nil)
}

func (c *HTTPClient) GetPayment(ctx context.Context, paymentID string) (Payment, error) {
	var out struct {
		ID     string `json:"id"`
		Status string `json:"status"`
	}
	if err := c.do(ctx, http.MethodGet, "/payments/"+paymentID, nil, &out); err != nil {
		return Payment{}, err
	}
	return Payment{ID: out.ID, Status: out.Status}, nil
}
