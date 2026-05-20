package worker

import (
	"crypto/tls"
	"errors"
	"fmt"
	"mime"
	"net"
	"net/mail"
	"net/smtp"
	"os"
	"strings"
	"time"
)

const smtpDialTimeout = 10 * time.Second

var errSMTPNotConfigured = errors.New("smtp is not configured")

type SMTPConfig struct {
	Host     string
	Port     string
	Username string
	Password string
	From     string
}

func LoadSMTPConfigFromEnv() SMTPConfig {
	return SMTPConfig{
		Host:     os.Getenv("SMTP_HOST"),
		Port:     os.Getenv("SMTP_PORT"),
		Username: os.Getenv("SMTP_USERNAME"),
		Password: os.Getenv("SMTP_PASSWORD"),
		From:     os.Getenv("SMTP_FROM"),
	}
}

func (c SMTPConfig) Configured() bool {
	c = c.normalized()
	return c.Host != "" && c.Port != "" && c.From != ""
}

func (c SMTPConfig) normalized() SMTPConfig {
	return SMTPConfig{
		Host:     strings.TrimSpace(c.Host),
		Port:     strings.TrimSpace(c.Port),
		Username: strings.TrimSpace(c.Username),
		Password: strings.TrimSpace(c.Password),
		From:     strings.TrimSpace(c.From),
	}
}

func sendEmail(cfg SMTPConfig, to string, subject string, body string) error {
	cfg = cfg.normalized()
	if !cfg.Configured() {
		return errSMTPNotConfigured
	}

	fromAddress, err := mail.ParseAddress(cfg.From)
	if err != nil {
		return fmt.Errorf("parse smtp from address: %w", err)
	}

	toAddress, err := mail.ParseAddress(strings.TrimSpace(to))
	if err != nil {
		return fmt.Errorf("parse recipient address: %w", err)
	}

	message := buildEmailMessage(fromAddress, toAddress, subject, body)
	address := net.JoinHostPort(cfg.Host, cfg.Port)
	recipients := []string{toAddress.Address}

	if cfg.Port == "465" {
		return sendEmailImplicitTLS(cfg, address, fromAddress.Address, recipients, message)
	}

	return sendEmailWithSTARTTLS(cfg, address, fromAddress.Address, recipients, message)
}

func sendEmailWithSTARTTLS(cfg SMTPConfig, address string, from string, recipients []string, message []byte) error {
	conn, err := dialSMTP(address)
	if err != nil {
		return fmt.Errorf("dial smtp server: %w", err)
	}

	client, err := smtp.NewClient(conn, cfg.Host)
	if err != nil {
		conn.Close()
		return fmt.Errorf("create smtp client: %w", err)
	}
	defer client.Close()

	if ok, _ := client.Extension("STARTTLS"); ok {
		if err := client.StartTLS(tlsConfig(cfg.Host)); err != nil {
			return fmt.Errorf("start smtp tls: %w", err)
		}
	}

	return sendSMTPMail(client, cfg, from, recipients, message)
}

func sendEmailImplicitTLS(cfg SMTPConfig, address string, from string, recipients []string, message []byte) error {
	conn, err := dialSMTP(address)
	if err != nil {
		return fmt.Errorf("dial smtp server: %w", err)
	}

	tlsConn := tls.Client(conn, tlsConfig(cfg.Host))
	if err := tlsConn.Handshake(); err != nil {
		conn.Close()
		return fmt.Errorf("smtp tls handshake: %w", err)
	}

	client, err := smtp.NewClient(tlsConn, cfg.Host)
	if err != nil {
		tlsConn.Close()
		return fmt.Errorf("create smtp tls client: %w", err)
	}
	defer client.Close()

	return sendSMTPMail(client, cfg, from, recipients, message)
}

func dialSMTP(address string) (net.Conn, error) {
	dialer := net.Dialer{Timeout: smtpDialTimeout}
	return dialer.Dial("tcp", address)
}

func sendSMTPMail(client *smtp.Client, cfg SMTPConfig, from string, recipients []string, message []byte) error {
	auth, err := smtpAuth(cfg)
	if err != nil {
		return err
	}
	if auth != nil {
		if err := client.Auth(auth); err != nil {
			return fmt.Errorf("smtp auth: %w", err)
		}
	}

	if err := client.Mail(from); err != nil {
		return fmt.Errorf("smtp mail from: %w", err)
	}
	for _, recipient := range recipients {
		if err := client.Rcpt(recipient); err != nil {
			return fmt.Errorf("smtp recipient %q: %w", recipient, err)
		}
	}

	writer, err := client.Data()
	if err != nil {
		return fmt.Errorf("smtp data: %w", err)
	}
	if _, err := writer.Write(message); err != nil {
		writer.Close()
		return fmt.Errorf("write smtp message: %w", err)
	}
	if err := writer.Close(); err != nil {
		return fmt.Errorf("close smtp message: %w", err)
	}
	if err := client.Quit(); err != nil {
		return fmt.Errorf("smtp quit: %w", err)
	}

	return nil
}

func smtpAuth(cfg SMTPConfig) (smtp.Auth, error) {
	if cfg.Username == "" && cfg.Password == "" {
		return nil, nil
	}
	if cfg.Username == "" || cfg.Password == "" {
		return nil, errors.New("smtp username and password must be configured together")
	}

	return smtp.PlainAuth("", cfg.Username, cfg.Password, cfg.Host), nil
}

func tlsConfig(host string) *tls.Config {
	return &tls.Config{
		ServerName: strings.TrimSpace(host),
		MinVersion: tls.VersionTLS12,
	}
}

func buildEmailMessage(from *mail.Address, to *mail.Address, subject string, body string) []byte {
	safeSubject := strings.NewReplacer("\r", " ", "\n", " ").Replace(subject)
	normalizedBody := strings.ReplaceAll(body, "\r\n", "\n")
	normalizedBody = strings.ReplaceAll(normalizedBody, "\r", "\n")
	normalizedBody = strings.ReplaceAll(normalizedBody, "\n", "\r\n")

	headers := []string{
		"From: " + from.String(),
		"To: " + to.String(),
		"Subject: " + mime.QEncoding.Encode("utf-8", safeSubject),
		"MIME-Version: 1.0",
		`Content-Type: text/plain; charset="utf-8"`,
		"Content-Transfer-Encoding: 8bit",
	}

	return []byte(strings.Join(headers, "\r\n") + "\r\n\r\n" + normalizedBody + "\r\n")
}
