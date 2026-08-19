package main

import (
	"strings"
	"testing"
)

func TestSMTPConfigRejectsMissingHostAsPermanent(t *testing.T) {
	err := (smtpConfig{Port: 587, Encryption: "starttls"}).validate()
	if err == nil || err.classification != classificationPermanent || err.reason != "smtp.host_required" {
		t.Fatalf("unexpected validation error: %#v", err)
	}
}

func TestBuildMessageCreatesMultipartAlternative(t *testing.T) {
	raw, err := buildMessage(smtpConfig{FromAddress: "noreply@example.com", FromName: "SForum"}, mailRequest{
		To: []string{"member@example.com"}, Subject: "提及通知", TextBody: "plain", HTMLBody: "<p>html</p>",
	})
	if err != nil {
		t.Fatal(err)
	}
	text := string(raw)
	for _, expected := range []string{"multipart/alternative", "text/plain", "text/html", "=?UTF-8?b?", "To: member@example.com"} {
		if !strings.Contains(text, expected) {
			t.Fatalf("message missing %q:\n%s", expected, text)
		}
	}
}

func TestClassifyNetworkFailureAsTemporary(t *testing.T) {
	err := classifySendError(assertionError("connection refused"))
	if err.classification != classificationTemporary || err.reason != "smtp.transport_failed" {
		t.Fatalf("unexpected classification: %#v", err)
	}
}

type assertionError string

func (e assertionError) Error() string { return string(e) }
