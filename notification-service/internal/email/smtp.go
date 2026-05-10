package email

import (
	"context"
	"fmt"
	"net/smtp"
)

type SMTPConfig struct {
	Host     string
	Port     string
	Username string
	Password string
	From     string
}

type SMTPProvider struct {
	cfg SMTPConfig
}

func NewSMTPProvider(cfg SMTPConfig) *SMTPProvider {
	return &SMTPProvider{cfg: cfg}
}

func (p *SMTPProvider) Send(_ context.Context, msg Message) error {
	addr := p.cfg.Host + ":" + p.cfg.Port
	auth := smtp.PlainAuth("", p.cfg.Username, p.cfg.Password, p.cfg.Host)
	body := fmt.Sprintf(
		"Subject: Payment Confirmed\r\n\r\nHi, your order #%s for $%d has been confirmed.",
		msg.OrderID, msg.Amount,
	)
	return smtp.SendMail(addr, auth, p.cfg.From, []string{msg.To}, []byte(body))
}
