package smtp

import (
	"context"
	"fmt"
	"io"
	"log"

	"github.com/emersion/go-smtp"
)

// Backend implements SMTP server backend
type Backend struct {
	handler MessageHandler
}

// MessageHandler processes received emails
type MessageHandler interface {
	HandleMessage(ctx context.Context, from string, to []string, data []byte) error
}

// NewBackend creates a new SMTP backend
func NewBackend(handler MessageHandler) *Backend {
	return &Backend{handler: handler}
}

// NewSession creates a new SMTP session
func (b *Backend) NewSession(c *smtp.Conn) (smtp.Session, error) {
	return &Session{
		backend: b,
		ctx:     context.Background(),
	}, nil
}

// Session represents an SMTP session
type Session struct {
	backend *Backend
	ctx     context.Context
	from    string
	to      []string
}

// AuthPlain handles PLAIN authentication (not implemented)
func (s *Session) AuthPlain(username, password string) error {
	return nil
}

// Mail handles MAIL FROM command
func (s *Session) Mail(from string, opts *smtp.MailOptions) error {
	log.Printf("MAIL FROM: %s", from)
	s.from = from
	return nil
}

// Rcpt handles RCPT TO command
func (s *Session) Rcpt(to string, opts *smtp.RcptOptions) error {
	log.Printf("RCPT TO: %s", to)
	s.to = append(s.to, to)
	return nil
}

// Data handles DATA command
func (s *Session) Data(r io.Reader) error {
	log.Printf("Receiving message data from %s to %v", s.from, s.to)

	data, err := io.ReadAll(r)
	if err != nil {
		log.Printf("Error reading message data: %v", err)
		return fmt.Errorf("failed to read message data: %w", err)
	}

	log.Printf("Received %d bytes of data", len(data))

	if err := s.backend.handler.HandleMessage(s.ctx, s.from, s.to, data); err != nil {
		log.Printf("Error handling message: %v", err)
		return fmt.Errorf("failed to handle message: %w", err)
	}

	log.Printf("Message handled successfully")
	return nil
}

// Reset resets the session state
func (s *Session) Reset() {
	s.from = ""
	s.to = nil
}

// Logout handles session logout
func (s *Session) Logout() error {
	return nil
}
