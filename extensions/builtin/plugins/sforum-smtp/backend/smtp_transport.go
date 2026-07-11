package main

import (
	"bytes"
	"crypto/tls"
	"fmt"
	"mime"
	"mime/multipart"
	"mime/quotedprintable"
	"net"
	"net/mail"
	"net/smtp"
	"strconv"
	"strings"
	"time"

	extensionsruntime "github.com/zhuchunshu/sforum/apps/api/app/Support/Extensions"
)

const (
	classificationTemporary = "temporary"
	classificationPermanent = "permanent"
)

type providerError struct{ classification, reason, message string }

func (e *providerError) Error() string { return e.message }

type smtpConfig struct {
	Host, Username, Password, Encryption, FromAddress, FromName string
	Port                                                        int
}

func (c smtpConfig) validate() *providerError {
	if strings.TrimSpace(c.Host) == "" {
		return &providerError{classificationPermanent, "smtp.host_required", "SMTP host is required."}
	}
	if c.Port < 1 || c.Port > 65535 {
		return &providerError{classificationPermanent, "smtp.port_invalid", "SMTP port is invalid."}
	}
	if _, err := mail.ParseAddress(c.FromAddress); err != nil {
		return &providerError{classificationPermanent, "smtp.from_invalid", "Sender address is invalid."}
	}
	switch c.Encryption {
	case "none", "starttls", "tls":
	default:
		return &providerError{classificationPermanent, "smtp.encryption_invalid", "SMTP encryption is invalid."}
	}
	return nil
}

func classifySendError(err error) *providerError {
	return &providerError{classificationTemporary, "smtp.transport_failed", err.Error()}
}

func buildMessage(config smtpConfig, request extensionsruntime.MailProviderRequest) ([]byte, error) {
	var body bytes.Buffer
	boundary := multipart.NewWriter(&body)
	from := (&mail.Address{Name: config.FromName, Address: config.FromAddress}).String()
	fmt.Fprintf(&body, "From: %s\r\nTo: %s\r\nSubject: %s\r\nMIME-Version: 1.0\r\nContent-Type: multipart/alternative; boundary=%q\r\n\r\n", from, strings.Join(request.To, ", "), mime.BEncoding.Encode("UTF-8", request.Subject), boundary.Boundary())
	writePart := func(contentType, value string) error {
		header := make(map[string][]string)
		header["Content-Type"] = []string{contentType + "; charset=UTF-8"}
		header["Content-Transfer-Encoding"] = []string{"quoted-printable"}
		part, err := boundary.CreatePart(header)
		if err != nil {
			return err
		}
		writer := quotedprintable.NewWriter(part)
		if _, err = writer.Write([]byte(value)); err != nil {
			return err
		}
		return writer.Close()
	}
	if err := writePart("text/plain", request.TextBody); err != nil {
		return nil, err
	}
	if request.HTMLBody != "" {
		if err := writePart("text/html", request.HTMLBody); err != nil {
			return nil, err
		}
	}
	if err := boundary.Close(); err != nil {
		return nil, err
	}
	return body.Bytes(), nil
}

func sendSMTP(config smtpConfig, request extensionsruntime.MailProviderRequest) *providerError {
	if err := config.validate(); err != nil {
		return err
	}
	raw, err := buildMessage(config, request)
	if err != nil {
		return &providerError{classificationPermanent, "smtp.message_invalid", err.Error()}
	}
	address := net.JoinHostPort(config.Host, strconv.Itoa(config.Port))
	if config.Encryption == "tls" {
		conn, err := tls.DialWithDialer(&net.Dialer{Timeout: 10 * time.Second}, "tcp", address, &tls.Config{ServerName: config.Host, MinVersion: tls.VersionTLS12})
		if err != nil {
			return classifySendError(err)
		}
		client, err := smtp.NewClient(conn, config.Host)
		if err != nil {
			return classifySendError(err)
		}
		defer client.Close()
		return deliver(client, config, request.To, raw)
	}
	client, err := smtp.Dial(address)
	if err != nil {
		return classifySendError(err)
	}
	defer client.Close()
	if config.Encryption == "starttls" {
		if err := client.StartTLS(&tls.Config{ServerName: config.Host, MinVersion: tls.VersionTLS12}); err != nil {
			return classifySendError(err)
		}
	}
	return deliver(client, config, request.To, raw)
}

func deliver(client *smtp.Client, config smtpConfig, recipients []string, raw []byte) *providerError {
	if config.Username != "" {
		if err := client.Auth(smtp.PlainAuth("", config.Username, config.Password, config.Host)); err != nil {
			return &providerError{classificationPermanent, "smtp.authentication_failed", err.Error()}
		}
	}
	if err := client.Mail(config.FromAddress); err != nil {
		return classifySendError(err)
	}
	for _, recipient := range recipients {
		if err := client.Rcpt(recipient); err != nil {
			return &providerError{classificationPermanent, "smtp.recipient_rejected", err.Error()}
		}
	}
	writer, err := client.Data()
	if err != nil {
		return classifySendError(err)
	}
	if _, err = writer.Write(raw); err != nil {
		return classifySendError(err)
	}
	if err = writer.Close(); err != nil {
		return classifySendError(err)
	}
	if err = client.Quit(); err != nil {
		return classifySendError(err)
	}
	return nil
}
