package mails

import (
	"crypto/tls"
	"errors"
	"fmt"
	"net"
	"net/smtp"
	"strconv"
	"strings"

	"github.com/MelloB1989/karma/config"
	"github.com/MelloB1989/karma/internal/aws"
	"github.com/MelloB1989/karma/models"
)

type MailServices string

const (
	AWS_SES  MailServices = "AWS_SES"
	SMTP     MailServices = "SMTP"
	MAILGUN  MailServices = "MAILGUN"
	SENDGRID MailServices = "SENDGRID"
)

type SMTPConfig struct {
	Host     string // e.g. "smtp.gmail.com"
	Port     int    // e.g. 587
	Username string // SMTP username
	Password string // SMTP password / app password
	UseTLS   bool   // If true, do TLS immediately (SMTPS / typically 465)
}

type MailClient struct {
	FromMail string
	Service  MailServices

	SMTP *SMTPConfig
}

func NewKarmaMail(fromMail string, service MailServices) *MailClient {
	m := &MailClient{
		FromMail: fromMail,
		Service:  service,
	}

	if service == SMTP {
		host := strings.TrimSpace(config.GetEnvRaw("SMTP_HOST"))
		portRaw := strings.TrimSpace(config.GetEnvRaw("SMTP_PORT"))
		username := config.GetEnvRaw("SMTP_USERNAME")
		password := config.GetEnvRaw("SMTP_PASSWORD")
		useTLSRaw := strings.TrimSpace(config.GetEnvRaw("SMTP_USE_TLS"))

		var port int
		if portRaw != "" {
			if p, err := strconv.Atoi(portRaw); err == nil {
				port = p
			}
		}

		m.SMTP = &SMTPConfig{
			Host:     host,
			Port:     port,
			Username: username,
			Password: password,
			UseTLS:   parseEnvBool(useTLSRaw),
		}
	}

	return m
}

func (m *MailClient) SendSingleMail(email models.SingleEmailRequest) error {
	if m.Service == AWS_SES {
		return aws.SendEmailToSingleRecipient(email, m.FromMail)
	} else if m.Service == SMTP {
		return sendSMTP(email, m.FromMail, m.SMTP)
	} else if m.Service == MAILGUN {
		return errors.New("Mailgun not supported yet")
	} else if m.Service == SENDGRID {
		return errors.New("Sendgrid not supported yet")
	} else {
		return errors.New("Mail service not supported")
	}
}

func parseEnvBool(v string) bool {
	switch strings.ToLower(strings.TrimSpace(v)) {
	case "1", "true", "yes", "y", "on":
		return true
	default:
		return false
	}
}

func sendSMTP(email models.SingleEmailRequest, from string, cfg *SMTPConfig) error {
	if cfg == nil {
		return errors.New("SMTP config is required")
	}
	if strings.TrimSpace(cfg.Host) == "" {
		return errors.New("SMTP host is required")
	}
	if cfg.Port <= 0 {
		return errors.New("SMTP port is required")
	}

	to := strings.TrimSpace(email.To)
	if to == "" {
		return errors.New("recipient email is required")
	}

	addr := net.JoinHostPort(cfg.Host, strconv.Itoa(cfg.Port))

	subject := email.Subject
	body := strings.TrimSpace(email.Body.Text)
	if body == "" {
		body = strings.TrimSpace(email.Body.HTML)
	}

	msg := strings.Join([]string{
		fmt.Sprintf("From: %s", from),
		fmt.Sprintf("To: %s", to),
		fmt.Sprintf("Subject: %s", subject),
		"MIME-Version: 1.0",
		"Content-Type: text/plain; charset=utf-8",
		"",
		body,
		"",
	}, "\r\n")

	var c *smtp.Client
	var err error

	if cfg.UseTLS {
		conn, dialErr := tls.Dial("tcp", addr, &tls.Config{
			ServerName: cfg.Host,
		})
		if dialErr != nil {
			return dialErr
		}
		c, err = smtp.NewClient(conn, cfg.Host)
		if err != nil {
			_ = conn.Close()
			return err
		}
	} else {
		conn, dialErr := net.Dial("tcp", addr)
		if dialErr != nil {
			return dialErr
		}
		c, err = smtp.NewClient(conn, cfg.Host)
		if err != nil {
			_ = conn.Close()
			return err
		}
		// Upgrade to TLS if offered (STARTTLS) when not doing implicit TLS.
		if ok, _ := c.Extension("STARTTLS"); ok {
			if err := c.StartTLS(&tls.Config{ServerName: cfg.Host}); err != nil {
				_ = c.Quit()
				return err
			}
		}
	}
	defer func() { _ = c.Quit() }()

	// Authenticate if credentials are provided. Many servers require auth.
	if strings.TrimSpace(cfg.Username) != "" || strings.TrimSpace(cfg.Password) != "" {
		auth := smtp.PlainAuth("", cfg.Username, cfg.Password, cfg.Host)
		if err := c.Auth(auth); err != nil {
			return err
		}
	}

	if err := c.Mail(from); err != nil {
		return err
	}
	if err := c.Rcpt(to); err != nil {
		return err
	}

	w, err := c.Data()
	if err != nil {
		return err
	}
	if _, err := w.Write([]byte(msg)); err != nil {
		_ = w.Close()
		return err
	}
	return w.Close()
}
