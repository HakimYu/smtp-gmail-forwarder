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

	for key, values := range msg.Header {
		if strings.ToLower(key) == "from" {
			fmt.Fprintf(&buf, "From: %s\r\n", from)
		} else if strings.ToLower(key) == "to" {
			fmt.Fprintf(&buf, "To: %s\r\n", strings.Join(to, ", "))
		} else {
			for _, value := range values {
				fmt.Fprintf(&buf, "%s: %s\r\n", key, value)
			}
		}
	}

	if msg.Header.Get("From") == "" {
		fmt.Fprintf(&buf, "From: %s\r\n", from)
	}
	if msg.Header.Get("To") == "" {
		fmt.Fprintf(&buf, "To: %s\r\n", strings.Join(to, ", "))
	}

	buf.WriteString("\r\n")

	if _, err := io.Copy(&buf, msg.Body); err != nil {
		return nil, fmt.Errorf("failed to copy body: %w", err)
	}

	return buf.Bytes(), nil
}
