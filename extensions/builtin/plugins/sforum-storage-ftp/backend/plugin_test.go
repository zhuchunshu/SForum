package main

import "testing"

func TestFTPConfigRequiresCredentials(t *testing.T) {
	t.Setenv("SFORUM_SETTING_HOST", "files.example.test")
	t.Setenv("SFORUM_SETTING_USERNAME", "forum")
	t.Setenv("SFORUM_SETTING_PASSWORD", "")
	if _, err := ftpConfigFromEnv(); err == nil {
		t.Fatal("expected missing password to be rejected")
	}
}

func TestFTPConfigUsesSafeDefaults(t *testing.T) {
	t.Setenv("SFORUM_SETTING_HOST", "files.example.test")
	t.Setenv("SFORUM_SETTING_USERNAME", "forum")
	t.Setenv("SFORUM_SETTING_PASSWORD", "secret")
	config, err := ftpConfigFromEnv()
	if err != nil {
		t.Fatal(err)
	}
	if config.port != 21 || config.rootPath != "/" || !config.passive || config.explicitTLS {
		t.Fatalf("config = %#v", config)
	}
}
