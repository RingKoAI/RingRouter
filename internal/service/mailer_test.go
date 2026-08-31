package service

import (
	"strings"
	"testing"
)

func TestValidateAddressRejectsHeaderInjection(t *testing.T) {
	valid := []string{"user@example.com", "a.b+tag@example.co.uk"}
	for _, address := range valid {
		if err := validateAddress(address); err != nil {
			t.Errorf("validateAddress(%q) = %v, want valid", address, err)
		}
	}
	invalid := []string{
		"",
		"user@example.com\r\nBcc: attacker@example.com",
		"user@example.com\nX-Injected: true",
		"Display Name <user@example.com>",
		"user@example.com,attacker@example.com",
	}
	for _, address := range invalid {
		if err := validateAddress(address); err == nil {
			t.Errorf("validateAddress(%q) accepted unsafe address", address)
		}
	}
}

func TestBuildMessageCannotSplitHeaders(t *testing.T) {
	message := string(buildMessage(
		"sender@example.com",
		"recipient@example.com",
		"hello\r\nBcc: attacker@example.com",
		"body\r\nignored",
	))
	if strings.Contains(message, "\r\nBcc:") {
		t.Fatal("subject created an injected SMTP header")
	}
	if !strings.Contains(message, "Subject: hello  Bcc: attacker@example.com\r\n") {
		t.Fatalf("subject was not safely flattened: %q", message)
	}
}
