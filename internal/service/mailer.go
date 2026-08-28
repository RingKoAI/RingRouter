// Package service hosts infrastructural services (mail delivery, ...).
package service

import (
	"crypto/tls"
	"fmt"
	"net"
	"net/smtp"
	"strconv"
	"strings"
	"time"

	"github.com/RingKoAI/RingRouter/internal/setting"
)

// mailTimeout bounds a single SMTP session.
const mailTimeout = 15 * time.Second

// Mailer sends transactional email over SMTP.
type Mailer struct {
	cfg func() setting.SMTPConfig
}

// NewMailer creates a Mailer reading its config lazily so runtime option
// changes take effect without restart.
func NewMailer() *Mailer {
	return &Mailer{cfg: func() setting.SMTPConfig { return setting.SMTP(decrypt) }}
}

// Send delivers a plain-text message using the persisted configuration.
func (m *Mailer) Send(to, subject, body string) error {
	return m.SendWith(m.cfg(), to, subject, body)
}

// SendWith delivers a plain-text message using an explicit configuration,
// which lets the setup wizard verify credentials before saving them.
func (m *Mailer) SendWith(cfg setting.SMTPConfig, to, subject, body string) error {
	if cfg.Host == "" || cfg.Port == 0 {
		return fmt.Errorf("SMTP is not configured")
	}
	if cfg.From == "" {
		return fmt.Errorf("SMTP sender address is not configured")
	}
	if cfg.Username != "" && cfg.Password == "" {
		return fmt.Errorf("SMTP password is required when username is set")
	}

	addr := net.JoinHostPort(cfg.Host, strconv.Itoa(cfg.Port))
	host := cfg.Host

	msg := buildMessage(cfg.From, to, subject, body)

	// Port 465 uses implicit TLS; everything else uses STARTTLS via smtp.SendMail.
	if cfg.Port == 465 {
		return sendTLS(addr, host, cfg, to, msg)
	}

	var auth smtp.Auth
	if cfg.Username != "" {
		auth = smtp.PlainAuth("", cfg.Username, cfg.Password, host)
	}
	conn, err := net.DialTimeout("tcp", addr, mailTimeout)
	if err != nil {
		return fmt.Errorf("connect %s: %w", addr, err)
	}
	// smtp.SendMail does not support dial timeouts, so wrap the pre-dialed conn.
	c, err := smtp.NewClient(conn, host)
	if err != nil {
		conn.Close()
		return fmt.Errorf("SMTP handshake: %w", err)
	}
	defer c.Close()

	if ok, _ := c.Extension("STARTTLS"); ok {
		if err := c.StartTLS(&tls.Config{ServerName: host}); err != nil {
			return fmt.Errorf("STARTTLS: %w", err)
		}
	}
	if auth != nil {
		if err := c.Auth(auth); err != nil {
			return fmt.Errorf("SMTP auth: %w", err)
		}
	}
	if err := c.Mail(cfg.From); err != nil {
		return fmt.Errorf("MAIL FROM: %w", err)
	}
	if err := c.Rcpt(to); err != nil {
		return fmt.Errorf("RCPT TO: %w", err)
	}
	w, err := c.Data()
	if err != nil {
		return fmt.Errorf("DATA: %w", err)
	}
	if _, err := w.Write(msg); err != nil {
		return fmt.Errorf("write body: %w", err)
	}
	if err := w.Close(); err != nil {
		return fmt.Errorf("close body: %w", err)
	}
	return c.Quit()
}

func sendTLS(addr, host string, cfg setting.SMTPConfig, to string, msg []byte) error {
	conn, err := tls.DialWithDialer(&net.Dialer{Timeout: mailTimeout}, "tcp", addr,
		&tls.Config{ServerName: host})
	if err != nil {
		return fmt.Errorf("connect %s (TLS): %w", addr, err)
	}
	defer conn.Close()

	c, err := smtp.NewClient(conn, host)
	if err != nil {
		return fmt.Errorf("SMTP handshake: %w", err)
	}
	defer c.Close()

	if cfg.Username != "" {
		if err := c.Auth(smtp.PlainAuth("", cfg.Username, cfg.Password, host)); err != nil {
			return fmt.Errorf("SMTP auth: %w", err)
		}
	}
	if err := c.Mail(cfg.From); err != nil {
		return fmt.Errorf("MAIL FROM: %w", err)
	}
	if err := c.Rcpt(to); err != nil {
		return fmt.Errorf("RCPT TO: %w", err)
	}
	w, err := c.Data()
	if err != nil {
		return fmt.Errorf("DATA: %w", err)
	}
	if _, err := w.Write(msg); err != nil {
		return fmt.Errorf("write body: %w", err)
	}
	if err := w.Close(); err != nil {
		return fmt.Errorf("close body: %w", err)
	}
	return c.Quit()
}

// decrypt is wired by the application bootstrap to avoid a service -> crypto
// package cycle; nil keeps passwords sealed.
var decrypt func(string) (string, error)

// SetDecryptor wires the secret-decryption function used for SMTP passwords.
func SetDecryptor(f func(string) (string, error)) {
	decrypt = f
}

func buildMessage(from, to, subject, body string) []byte {
	var b strings.Builder
	b.WriteString("From: " + from + "\r\n")
	b.WriteString("To: " + to + "\r\n")
	b.WriteString("Subject: " + subject + "\r\n")
	b.WriteString("MIME-Version: 1.0\r\n")
	b.WriteString("Content-Type: text/plain; charset=UTF-8\r\n")
	b.WriteString("\r\n")
	b.WriteString(body)
	return []byte(b.String())
}
