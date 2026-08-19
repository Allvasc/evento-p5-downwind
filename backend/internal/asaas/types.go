package asaas

type CustomerParams struct {
	Name  string
	Email string
	Phone string
	CPF   string
}

type ChargeParams struct {
	CustomerID  string
	BillingType string // "PIX"
	ValueCents  int
	Description string
	ExternalRef string // our order_number, for reconciliation
}

type Charge struct {
	PaymentID    string
	Status       string // PENDING, CONFIRMED, RECEIVED, OVERDUE...
	PixQRImage   string // base64 PNG, when BillingType=PIX
	PixCopyPaste string
}

type Payment struct {
	ID     string
	Status string
}

// WebhookEvent is the subset of the Asaas webhook payload we act on.
// https://docs.asaas.com/docs/webhook-eventos
type WebhookEvent struct {
	Event   string `json:"event"`
	Payment struct {
		ID          string  `json:"id"`
		Status      string  `json:"status"`
		Customer    string  `json:"customer"`
		Value       float64 `json:"value"`
		ExternalRef string  `json:"externalReference"`
	} `json:"payment"`
}
