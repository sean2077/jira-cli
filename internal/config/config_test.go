package config

import (
	"path/filepath"
	"testing"
)

func TestParseProfileConfig(t *testing.T) {
	file, err := ParseProfileConfig(`
default_profile = "private"

[profiles.private]
type = "server"
base_url = "https://jira.example.com"
user = "jira-user"
token_env = "JIRA_PRIVATE_TOKEN"
`)
	if err != nil {
		t.Fatalf("ParseProfileConfig returned error: %v", err)
	}
	if file.DefaultProfile != "private" {
		t.Fatalf("default profile = %q", file.DefaultProfile)
	}
	profile := file.Profiles["private"]
	if profile.Type != "server" || profile.BaseURL != "https://jira.example.com" || profile.User != "jira-user" || profile.TokenEnv != "JIRA_PRIVATE_TOKEN" {
		t.Fatalf("profile parsed incorrectly: %#v", profile)
	}
}

func TestParseProfileConfigRejectsPlaintextSecrets(t *testing.T) {
	_, err := ParseProfileConfig(`
[profiles.private]
password = "secret"
`)
	if err == nil {
		t.Fatal("expected plaintext secret rejection")
	}
}

func TestResolvePrecedence(t *testing.T) {
	profiles := ProfileFile{
		DefaultProfile: "private",
		Profiles: map[string]Profile{
			"private": {
				Type:     "server",
				BaseURL:  "https://profile.example.com",
				User:     "profile-user",
				TokenEnv: "JIRA_PROFILE_TOKEN",
			},
		},
	}
	env := map[string]string{
		"JIRA_BASE_URL":  "https://env.example.com",
		"JIRA_USER":      "env-user",
		"JIRA_API_TOKEN": "redacted",
	}
	flags := Values{
		BaseURL: "https://flag.example.com",
		User:    "flag-user",
	}

	resolved := Resolve(flags, env, profiles)
	if resolved.Profile != "private" {
		t.Fatalf("profile = %q", resolved.Profile)
	}
	if resolved.Type != "server" {
		t.Fatalf("type = %q", resolved.Type)
	}
	if resolved.BaseURL != "https://flag.example.com" {
		t.Fatalf("base URL = %q", resolved.BaseURL)
	}
	if resolved.User != "flag-user" {
		t.Fatalf("user = %q", resolved.User)
	}
	if resolved.TokenEnv != "JIRA_API_TOKEN" {
		t.Fatalf("token env = %q", resolved.TokenEnv)
	}
}

func TestResolveDoesNotPersistTokenValues(t *testing.T) {
	resolved := Resolve(Values{}, map[string]string{"JIRA_PASSWORD": "secret"}, ProfileFile{})
	if resolved.TokenEnv != "JIRA_PASSWORD" {
		t.Fatalf("token env = %q", resolved.TokenEnv)
	}
	if resolved.TokenEnv == "secret" {
		t.Fatal("resolved config stored secret value")
	}
}

func TestEnvMap(t *testing.T) {
	env := EnvMap([]string{"JIRA_BASE_URL=https://jira.example.com", "BROKEN", "JIRA_USER=agent"})
	if env["JIRA_BASE_URL"] != "https://jira.example.com" || env["JIRA_USER"] != "agent" {
		t.Fatalf("EnvMap parsed incorrectly: %#v", env)
	}
	if _, ok := env["BROKEN"]; ok {
		t.Fatalf("EnvMap retained invalid item: %#v", env)
	}
}

func TestValidateRequiresSecretByReference(t *testing.T) {
	values := Values{Type: "server", BaseURL: "https://jira.example.com", User: "agent", TokenEnv: "JIRA_TOKEN"}
	if got := Validate(values, map[string]string{}); len(got) != 1 || got[0] != "token" {
		t.Fatalf("Validate missing token = %#v, want [token]", got)
	}
	if got := Validate(values, map[string]string{"JIRA_TOKEN": "secret"}); len(got) != 0 {
		t.Fatalf("Validate with token = %#v, want empty", got)
	}
}

func TestLoadProfileConfigMissingFileIsEmpty(t *testing.T) {
	path := filepath.Join(t.TempDir(), "missing.toml")
	file, err := LoadProfileConfig(path)
	if err != nil {
		t.Fatalf("LoadProfileConfig returned error: %v", err)
	}
	if file.Profiles == nil || len(file.Profiles) != 0 {
		t.Fatalf("file = %#v, want empty profile map", file)
	}
}
