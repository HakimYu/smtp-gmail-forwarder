package smtp

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log"
	"net/mail"
	"strings"
)

// GmailSender sends emails via Gmail API
type GmailSender interface {
	SendMessage(ctx context.Context, rawMessage []byte) error
}

// ForwarderHandler forwards SMTP messages to Gmail
type ForwarderHandler struct {
	gmailClient GmailSender
}

// NewForwarderHandler creates a new forwarder handler
func NewForwarderHandler(gmailClient GmailSender) *ForwarderHandler {
	return &ForwarderHandler{
		gmailClient: gmailClient,
	}
}

// HandleMessage processes incoming SMTP message and forwards via Gmail
func (h *ForwarderHandler) HandleMessage(ctx context.Context, from string, to []string, data []byte) error {
	log.Printf("Forwarding message from %s to %v", from, to)

	message, err := h.buildRFC2822Message(from, to, data)
	if err != nil {
		return fmt.Errorf("failed to build message: %w", err)
	}

	if err := h.gmailClient.SendMessage(ctx, message); err != nil {
		return fmt.Errorf("failed to send via Gmail: %w", err)
	}

	log.Printf("Message forwarded successfully via Gmail")
	return nil
}

// buildRFC2822Message constructs proper RFC 2822 message
func (h *ForwarderHandler) buildRFC2822Message(from string, to []string, data []byte) ([]byte, error) {
	msg, err := mail.ReadMessage(bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("failed to parse message: %w", err)
	}

	var buf bytes.Buffer

	// Track which headers we've written to avoid duplicates
	hasFrom := false
	hasTo := false

	for key, values := range msg.Header {
		keyLower := strings.ToLower(key)
		if keyLower == "from" {
			fmt.Fprintf(&buf, "From: %s\r\n", from)
			hasFrom = true
		} else if keyLower == "to" {
			fmt.Fprintf(&buf, "To: %s\r\n", strings.Join(to, ", "))
			hasTo = true
		} else {
			for _, value := range values {
				fmt.Fprintf(&buf, "%s: %s\r\n", key, value)
			}
		}
	}

	// Only add headers if they weren't present in the original message
	if !hasFrom {
		fmt.Fprintf(&buf, "From: %s\r\n", from)
	}
	if !hasTo {
		fmt.Fprintf(&buf, "To: %s\r\n", strings.Join(to, ", "))
	}

	buf.WriteString("\r\n")

	if _, err := io.Copy(&buf, msg.Body); err != nil {
		return nil, fmt.Errorf("failed to copy body: %w", err)
	}

	return buf.Bytes(), nil
}
