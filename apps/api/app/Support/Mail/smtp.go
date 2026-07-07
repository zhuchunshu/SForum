package mail

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"net"
	"net/smtp"
	"strings"
)

const (
	ProviderSMTP       = "smtp"
	EncryptionNone     = "none"
	EncryptionStartTLS = "starttls"
	EncryptionTLS      = "tls"

	defaultSMTPPort = 587
)

// SMTPConfig 是 SMTP provider 的连接配置。
type SMTPConfig struct {
	Host        string
	Port        int
	Username    string
	Password    string
	Encryption  string // none / starttls / tls
	FromAddress string
	FromName    string
}

// SMTPProvider 使用标准库 net/smtp 投递邮件。
// 支持隐式 TLS、STARTTLS 与明文连接；生产建议使用 starttls 或 tls。
type SMTPProvider struct {
	config SMTPConfig
}

func NewSMTPProvider(config SMTPConfig) *SMTPProvider {
	if config.Port <= 0 {
		config.Port = defaultSMTPPort
	}
	encryption := strings.ToLower(strings.TrimSpace(config.Encryption))
	if encryption == "" {
		encryption = EncryptionStartTLS
	}
	config.Encryption = encryption
	return &SMTPProvider{config: config}
}

func (p *SMTPProvider) Name() string { return ProviderSMTP }

func (p *SMTPProvider) Send(_ context.Context, message Message) error {
	if strings.TrimSpace(message.To) == "" {
		return errors.New("mail: missing recipient")
	}
	if strings.TrimSpace(p.config.FromAddress) == "" {
		return errors.New("mail: missing from address")
	}
	addr := net.JoinHostPort(p.config.Host, fmt.Sprintf("%d", p.config.Port))
	rawMessage := p.buildRFC822(message)
	recipients := []string{message.To}

	switch p.config.Encryption {
	case EncryptionTLS:
		return p.sendTLS(addr, recipients, rawMessage)
	case EncryptionNone:
		return smtp.SendMail(addr, p.auth(), p.config.FromAddress, recipients, rawMessage)
	default: // starttls
		return smtp.SendMail(addr, p.auth(), p.config.FromAddress, recipients, rawMessage)
	}
}

func (p *SMTPProvider) auth() smtp.Auth {
	if p.config.Username == "" {
		return nil
	}
	return smtp.PlainAuth("", p.config.Username, p.config.Password, p.config.Host)
}

func (p *SMTPProvider) sendTLS(addr string, recipients []string, rawMessage []byte) error {
	// 隐式 TLS：先建立 TLS 连接，再用 smtp.NewClient 走标准对话。
	tlsConfig := &tls.Config{ServerName: p.config.Host}
	conn, err := tls.Dial("tcp", addr, tlsConfig)
	if err != nil {
		return fmt.Errorf("mail: tls dial: %w", err)
	}
	defer func() { _ = conn.Close() }()

	client, err := smtp.NewClient(conn, p.config.Host)
	if err != nil {
		return fmt.Errorf("mail: smtp client: %w", err)
	}
	defer func() { _ = client.Quit() }()

	if auth := p.auth(); auth != nil {
		if err := client.Auth(auth); err != nil {
			return fmt.Errorf("mail: smtp auth: %w", err)
		}
	}
	if err := client.Mail(p.config.FromAddress); err != nil {
		return fmt.Errorf("mail: smtp MAIL FROM: %w", err)
	}
	for _, to := range recipients {
		if err := client.Rcpt(to); err != nil {
			return fmt.Errorf("mail: smtp RCPT TO: %w", err)
		}
	}
	writer, err := client.Data()
	if err != nil {
		return fmt.Errorf("mail: smtp DATA: %w", err)
	}
	if _, err := writer.Write(rawMessage); err != nil {
		return fmt.Errorf("mail: smtp write body: %w", err)
	}
	return writer.Close()
}

// buildRFC822 构造符合 RFC 822 头部 + 正文的字节流。
func (p *SMTPProvider) buildRFC822(message Message) []byte {
	from := p.config.FromAddress
	if p.config.FromName != "" {
		from = fmt.Sprintf("%s <%s>", p.config.FromName, p.config.FromAddress)
	}
	headers := strings.Join([]string{
		fmt.Sprintf("From: %s", from),
		fmt.Sprintf("To: %s", message.To),
		fmt.Sprintf("Subject: %s", p.encodeHeader(message.Subject)),
		"MIME-Version: 1.0",
		"Content-Type: text/plain; charset=UTF-8",
		"Content-Transfer-Encoding: 8bit",
	}, "\r\n")
	return []byte(headers + "\r\n\r\n" + message.TextBody)
}

// encodeHeader 对含非 ASCII 字符的头部值做最小化兜底（v1 不引入 mime.Q 编码器复杂度）。
func (p *SMTPProvider) encodeHeader(value string) string {
	return value
}
