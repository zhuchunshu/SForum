package main

import "testing"

func TestSFTPConfigRequiresHostKeyFingerprint(t *testing.T) {
	t.Setenv("SFORUM_SETTING_HOST", "files.example.test")
	t.Setenv("SFORUM_SETTING_USERNAME", "forum")
	t.Setenv("SFORUM_SETTING_PASSWORD", "secret")
	if _, err := sftpConfigFromEnv(); err == nil {
		t.Fatal("expected missing host key fingerprint to be rejected")
	}
}

func TestSFTPConfigAllowsPasswordAuthentication(t *testing.T) {
	t.Setenv("SFORUM_SETTING_HOST", "files.example.test")
	t.Setenv("SFORUM_SETTING_USERNAME", "forum")
	t.Setenv("SFORUM_SETTING_PASSWORD", "secret")
	t.Setenv("SFORUM_SETTING_HOST_KEY_FINGERPRINT", "SHA256:server-key")
	config, err := sftpConfigFromEnv()
	if err != nil {
		t.Fatal(err)
	}
	if config.port != 22 || config.rootPath != "/" {
		t.Fatalf("config = %#v", config)
	}
}
