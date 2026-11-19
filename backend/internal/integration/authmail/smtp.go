package authmail

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"net"
	"net/mail"
	"net/smtp"
	"strings"
	"time"
)

type Sender struct {
	address   string
	host      string
	from      string
	username  string
	password  string
	plaintext bool
}

func NewSender(address, from, username, password string, localPlaintext bool) (*Sender, error) {
	host, port, err := net.SplitHostPort(address)
	if err != nil || host == "" || port == "" {
		return nil, errors.New("AUTH_SMTP_ADDR harus berformat host:port")
	}
	fromAddress, err := mail.ParseAddress(from)
	if err != nil || fromAddress.Address != from || strings.ContainsAny(from, "\r\n") {
		return nil, errors.New("AUTH_SMTP_FROM harus berupa alamat email")
	}
	if (username == "") != (password == "") {
		return nil, errors.New("AUTH_SMTP_USERNAME dan AUTH_SMTP_PASSWORD harus diisi bersama")
	}
	if localPlaintext && (!net.ParseIP(host).IsLoopback() || username != "") {
		return nil, errors.New("SMTP plaintext hanya tersedia untuk IP loopback tanpa kredensial SMTP")
	}
	return &Sender{address: address, host: host, from: from, username: username, password: password, plaintext: localPlaintext}, nil
}

func (s *Sender) Send(ctx context.Context, recipient, subject, body string) error {
	if strings.ContainsAny(recipient+subject, "\r\n") {
		return errors.New("invalid SMTP message headers")
	}
	dialer := net.Dialer{Timeout: 6 * time.Second}
	conn, err := dialer.DialContext(ctx, "tcp", s.address)
	if err != nil {
		return err
	}
	defer conn.Close()
	deadline := time.Now().Add(6 * time.Second)
	if limit, ok := ctx.Deadline(); ok && limit.Before(deadline) {
		deadline = limit
	}
	if err := conn.SetDeadline(deadline); err != nil {
		return err
	}

	client, err := smtp.NewClient(conn, s.host)
	if err != nil {
		return err
	}
	defer client.Close()
	if !s.plaintext {
		if err := client.StartTLS(&tls.Config{ServerName: s.host, MinVersion: tls.VersionTLS12}); err != nil {
			return err
		}
		if s.username != "" {
			if err := client.Auth(smtp.PlainAuth("", s.username, s.password, s.host)); err != nil {
				return err
			}
		}
	}
	if err := client.Mail(s.from); err != nil {
		return err
	}
	if err := client.Rcpt(recipient); err != nil {
		return err
	}
	data, err := client.Data()
	if err != nil {
		return err
	}
	if _, err := fmt.Fprintf(data, "From: %s\r\nTo: %s\r\nSubject: %s\r\nMIME-Version: 1.0\r\nContent-Type: text/plain; charset=UTF-8\r\n\r\n%s",
		s.from, recipient, subject, strings.ReplaceAll(body, "\n", "\r\n")); err != nil {
		return err
	}
	return data.Close()
}
