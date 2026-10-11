package services

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/spark-school/backend/internal/config"
)

func generateIdempotenceKey() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	b[6] = (b[6] & 0x0f) | 0x40 // Version 4
	b[8] = (b[8] & 0x3f) | 0x80 // Variant RFC4122
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:])
}

type YooKassaService struct {
	cfg        *config.Config
	httpClient *http.Client
	baseURL    string
}

func NewYooKassaService(cfg *config.Config) *YooKassaService {
	return &YooKassaService{
		cfg: cfg,
		httpClient: &http.Client{
			Timeout: 15 * time.Second,
		},
		baseURL: "https://api.yookassa.ru/v3",
	}
}

// YooKassaAmount represents monetary value and currency.
type YooKassaAmount struct {
	Value    string `json:"value"`
	Currency string `json:"currency"`
}

// YooKassaConfirmation represents redirect configuration.
type YooKassaConfirmation struct {
	Type            string `json:"type"`
	ReturnURL       string `json:"return_url,omitempty"`
	ConfirmationURL string `json:"confirmation_url,omitempty"`
}

// YooKassaCustomer represents receipt customer info for 54-FZ.
type YooKassaCustomer struct {
	Email string `json:"email,omitempty"`
	Phone string `json:"phone,omitempty"`
}

// YooKassaReceiptItem represents line item in 54-FZ fiscal receipt.
type YooKassaReceiptItem struct {
	Description    string         `json:"description"`
	Quantity       string         `json:"quantity"`
	Amount         YooKassaAmount `json:"amount"`
	VatCode        int            `json:"vat_code"` // 1: без НДС (для самозанятых и УСН)
	PaymentMode    string         `json:"payment_mode,omitempty"` // "full_payment"
	PaymentSubject string         `json:"payment_subject,omitempty"` // "service"
}

// YooKassaReceipt represents 54-FZ fiscal receipt data.
type YooKassaReceipt struct {
	Customer YooKassaCustomer     `json:"customer"`
	Items    []YooKassaReceiptItem `json:"items"`
}

// YooKassaPaymentRequest payload for POST /v3/payments.
type YooKassaPaymentRequest struct {
	Amount       YooKassaAmount         `json:"amount"`
	Capture      bool                   `json:"capture"`
	Confirmation YooKassaConfirmation   `json:"confirmation"`
	Description  string                 `json:"description"`
	Metadata     map[string]string      `json:"metadata,omitempty"`
	Receipt      *YooKassaReceipt       `json:"receipt,omitempty"`
}

// YooKassaPaymentResponse response from YooKassa API.
type YooKassaPaymentResponse struct {
	ID           string                 `json:"id"`
	Status       string                 `json:"status"` // "pending", "waiting_for_capture", "succeeded", "canceled"
	Paid         bool                   `json:"paid"`
	Amount       YooKassaAmount         `json:"amount"`
	Confirmation YooKassaConfirmation   `json:"confirmation"`
	Description  string                 `json:"description"`
	Metadata     map[string]string      `json:"metadata"`
	CreatedAt    string                 `json:"created_at"`
}

// CreatePayment creates an official invoice in YooKassa and returns confirmation_url for checkout.
func (s *YooKassaService) CreatePayment(ctx context.Context, req YooKassaPaymentRequest) (*YooKassaPaymentResponse, error) {
	if s.cfg.YooKassaShopID == "" || s.cfg.YooKassaSecretKey == "" {
		return nil, fmt.Errorf("yookassa credentials are not configured")
	}

	resp, err := s.doCreatePayment(ctx, req)
	if err != nil && req.Receipt == nil && (strings.Contains(err.Error(), "receipt") || strings.Contains(err.Error(), "400")) {
		// Shop requires 54-FZ fiscal receipt; auto-construct compliant receipt for self-employed/USN
		customerEmail := "student@so-called-spark.ru"
		if req.Metadata != nil && req.Metadata["email"] != "" {
			customerEmail = req.Metadata["email"]
		} else if req.Metadata != nil && req.Metadata["username"] != "" {
			customerEmail = req.Metadata["username"] + "@t.me"
		}

		reqWithReceipt := req
		reqWithReceipt.Receipt = &YooKassaReceipt{
			Customer: YooKassaCustomer{Email: customerEmail},
			Items: []YooKassaReceiptItem{
				{
					Description:    req.Description,
					Quantity:       "1.00",
					Amount:         req.Amount,
					VatCode:        1, // 1: без НДС
					PaymentMode:    "full_payment",
					PaymentSubject: "service",
				},
			},
		}
		return s.doCreatePayment(ctx, reqWithReceipt)
	}

	return resp, err
}

func (s *YooKassaService) doCreatePayment(ctx context.Context, req YooKassaPaymentRequest) (*YooKassaPaymentResponse, error) {
	bodyBytes, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal payment request: %w", err)
	}

	url := s.baseURL + "/payments"
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(bodyBytes))
	if err != nil {
		return nil, fmt.Errorf("failed to create http request: %w", err)
	}

	idempotenceKey := generateIdempotenceKey()
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Idempotence-Key", idempotenceKey)
	httpReq.SetBasicAuth(s.cfg.YooKassaShopID, s.cfg.YooKassaSecretKey)

	resp, err := s.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("yookassa request failed: %w", err)
	}
	defer resp.Body.Close()

	respBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("yookassa api error (status %d): %s", resp.StatusCode, string(respBytes))
	}

	var paymentResp YooKassaPaymentResponse
	if err := json.Unmarshal(respBytes, &paymentResp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal payment response: %w", err)
	}

	return &paymentResp, nil
}

// GetPayment fetches payment details directly from YooKassa to verify webhook authenticity.
func (s *YooKassaService) GetPayment(ctx context.Context, paymentID string) (*YooKassaPaymentResponse, error) {
	cleanID := strings.TrimSpace(paymentID)
	if cleanID == "" {
		return nil, fmt.Errorf("empty payment id")
	}
	if s.cfg.YooKassaShopID == "" || s.cfg.YooKassaSecretKey == "" {
		return nil, fmt.Errorf("yookassa credentials are not configured")
	}

	url := fmt.Sprintf("%s/payments/%s", s.baseURL, cleanID)
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create http request: %w", err)
	}

	httpReq.SetBasicAuth(s.cfg.YooKassaShopID, s.cfg.YooKassaSecretKey)

	resp, err := s.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("yookassa request failed: %w", err)
	}
	defer resp.Body.Close()

	respBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("yookassa get payment error (status %d): %s", resp.StatusCode, string(respBytes))
	}

	var paymentResp YooKassaPaymentResponse
	if err := json.Unmarshal(respBytes, &paymentResp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal payment response: %w", err)
	}

	return &paymentResp, nil
}
