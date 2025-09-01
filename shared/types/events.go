package types

import "time"

type EventType string

const (
	EventPaymentInitiated   EventType = "payment.initiated"
	EventInvoiceValidated   EventType = "invoice.validated"
	EventWalletDebited      EventType = "wallet.debited"
	EventPaymentProcessed   EventType = "payment.processed"
	EventPaymentFailed      EventType = "payment.failed"
	EventRefundInitiated    EventType = "refund.initiated"
	EventWalletCredited     EventType = "wallet.credited"
	EventRefundCompleted    EventType = "refund.completed"
)

type PaymentEvent struct {
	ID            string                 `json:"id" dynamodb:"ID"`
	PaymentID     string                 `json:"paymentId" dynamodb:"PaymentID"`
	UserID        string                 `json:"userId" dynamodb:"UserID"`
	EventType     string                 `json:"eventType" dynamodb:"EventType"`
	Amount        float64                `json:"amount" dynamodb:"Amount"`
	Status        string                 `json:"status" dynamodb:"Status"`
	Metadata      map[string]interface{} `json:"metadata,omitempty" dynamodb:"Metadata,omitempty"`
	CorrelationID string                 `json:"correlationId" dynamodb:"CorrelationID"`
	Timestamp     time.Time              `json:"timestamp" dynamodb:"Timestamp"`
}