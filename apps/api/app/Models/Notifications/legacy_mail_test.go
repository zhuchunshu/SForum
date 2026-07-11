package notifications

import "testing"

func TestLegacyMailSettingsPreservePluginValues(t *testing.T) {
	legacy := map[string]string{
		"mail.provider": "smtp", "mail.from_address": "old@example.com", "mail.from_name": "Old",
		"mail.smtp.host": "smtp.old.example", "mail.smtp.port": "465", "mail.smtp.username": "old-user",
		"mail.smtp.password": "old-secret", "mail.smtp.encryption": "tls",
	}
	current := map[string]string{"host": "smtp.new.example", "password": "new-secret"}
	settings, selectSMTP := legacyMailSettings(legacy, current)
	if !selectSMTP {
		t.Fatal("legacy smtp should select builtin plugin")
	}
	if settings["host"] != "smtp.new.example" || settings["password"] != "new-secret" {
		t.Fatalf("new plugin values overwritten: %#v", settings)
	}
	if settings["port"] != "465" || settings["from_address"] != "old@example.com" {
		t.Fatalf("legacy values not adopted: %#v", settings)
	}
}

func TestLegacyNonSMTPLeavesProviderUnconfigured(t *testing.T) {
	for _, provider := range []string{"", "noop", "dev_log", "unknown"} {
		_, selectSMTP := legacyMailSettings(map[string]string{"mail.provider": provider}, nil)
		if selectSMTP {
			t.Fatalf("provider %q selected smtp", provider)
		}
	}
}
