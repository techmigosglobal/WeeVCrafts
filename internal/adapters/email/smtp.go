package email

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"mime"
	"net"
	"net/mail"
	"net/smtp"
	"strings"
	"time"

	"github.com/wecratfs/commerce/internal/ports"
)

const sendTimeout = 30 * time.Second

// SMTP is a provider-neutral email adapter for account-action messages. It
// uses STARTTLS when the relay advertises it and keeps one-time action URLs
// inside the message body only; callers and logs never receive provider data.
type SMTP struct {
	address  string
	host     string
	username string
	password string
	from     string
}

func NewSMTP(address, username, password, from string) *SMTP {
	host, _, err := net.SplitHostPort(address)
	if err != nil {
		host = address
	}
	return &SMTP{address: address, host: host, username: username, password: password, from: from}
}

func (s *SMTP) Send(ctx context.Context, message ports.EmailMessage) error {
	if s == nil || s.address == "" || s.from == "" || !validHeader(s.from) || !validHeader(message.Subject) || !validHeader(message.ActionURL) {
		return errors.New("smtp sender is not configured")
	}
	recipient, err := mail.ParseAddress(message.To)
	if err != nil || recipient.Address != message.To || !validHeader(message.To) {
		return errors.New("smtp recipient is invalid")
	}
	from, err := mail.ParseAddress(s.from)
	if err != nil || from.Address != s.from || !validHeader(s.from) {
		return errors.New("smtp sender address is invalid")
	}

	sendContext, cancel := context.WithTimeout(ctx, sendTimeout)
	defer cancel()
	connection, err := (&net.Dialer{}).DialContext(sendContext, "tcp", s.address)
	if err != nil {
		return err
	}
	defer connection.Close()
	if deadline, ok := sendContext.Deadline(); ok {
		_ = connection.SetDeadline(deadline)
	}

	client, err := smtp.NewClient(connection, s.host)
	if err != nil {
		return err
	}
	defer client.Close()
	if ok, _ := client.Extension("STARTTLS"); ok {
		if err := client.StartTLS(&tls.Config{ServerName: s.host, MinVersion: tls.VersionTLS12}); err != nil {
			return err
		}
	}
	if s.username != "" {
		if err := client.Auth(smtp.PlainAuth("", s.username, s.password, s.host)); err != nil {
			return err
		}
	}
	if err := client.Mail(from.Address); err != nil {
		return err
	}
	if err := client.Rcpt(recipient.Address); err != nil {
		return err
	}
	writer, err := client.Data()
	if err != nil {
		return err
	}
	if _, err := writer.Write(formatMessage(from, recipient, message)); err != nil {
		_ = writer.Close()
		return err
	}
	if err := writer.Close(); err != nil {
		return err
	}
	return client.Quit()
}

func formatMessage(from, recipient *mail.Address, message ports.EmailMessage) []byte {
	fromHeader := from.String()
	recipientHeader := recipient.String()
	if message.DisplayName != "" {
		recipientHeader = (&mail.Address{Name: message.DisplayName, Address: recipient.Address}).String()
	}
	return []byte(fmt.Sprintf("From: %s\r\nTo: %s\r\nSubject: %s\r\nMIME-Version: 1.0\r\nContent-Type: text/plain; charset=UTF-8\r\n\r\nWeCratfs account action\r\n\r\nOpen this link to continue:\r\n%s\r\n\r\nIf you did not request this, you can ignore this email.\r\n", fromHeader, recipientHeader, mime.QEncoding.Encode("UTF-8", message.Subject), message.ActionURL))
}

func validHeader(value string) bool {
	return value != "" && !strings.ContainsAny(value, "\r\n")
}
