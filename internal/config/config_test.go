package config

import (
	"os"
	"path/filepath"
	"strings"
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

func TestParseProfileConfigAcceptsProfileTokenAndPassword(t *testing.T) {
	file, err := ParseProfileConfig(`
[profiles.private]
token = "profile-token"

[profiles.password]
password = "profile-password"
`)
	if err != nil {
		t.Fatalf("ParseProfileConfig returned error: %v", err)
	}
	if file.Profiles["private"].Token != "profile-token" {
		t.Fatalf("token = %q", file.Profiles["private"].Token)
	}
	if file.Profiles["password"].Password != "profile-password" {
		t.Fatalf("password = %q", file.Profiles["password"].Password)
	}
}

func TestParseProfileConfigNormalizesProfileNames(t *testing.T) {
	file, err := ParseProfileConfig(`
default_profile = "Private"

[profiles.Private]
type = "server"
base_url = "https://jira.example.com"
user = "agent"
token = "profile-token"
`)
	if err != nil {
		t.Fatalf("ParseProfileConfig returned error: %v", err)
	}
	if file.DefaultProfile != "private" {
		t.Fatalf("default profile = %q, want private", file.DefaultProfile)
	}
	if _, ok := file.Profiles["Private"]; ok {
		t.Fatalf("profile retained mixed-case key: %#v", file.Profiles)
	}
	if file.Profiles["private"].Token != "profile-token" {
		t.Fatalf("normalized profile = %#v", file.Profiles["private"])
	}
}

func TestParseProfileConfigRejectsCaseVariantProfileDuplicates(t *testing.T) {
	_, err := ParseProfileConfig(`
[profiles.Private]
token = "first"

[profiles.private]
token = "second"
`)
	if err == nil {
		t.Fatal("expected case-variant duplicate profile rejection")
	}
	if !strings.Contains(err.Error(), "conflicts") {
		t.Fatalf("err = %v, want conflict", err)
	}
}

func TestParseProfileConfigRejectsEmptyProfileName(t *testing.T) {
	_, err := ParseProfileConfig(`
[profiles."   "]
token = "secret"
`)
	if err == nil {
		t.Fatal("expected empty normalized profile name rejection")
	}
	if !strings.Contains(err.Error(), "must not be empty") {
		t.Fatalf("err = %v, want empty-name rejection", err)
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
	if resolved.Secret != "redacted" {
		t.Fatalf("secret = %q", resolved.Secret)
	}
	if resolved.SecretSource.Kind != SecretSourceEnv || resolved.SecretSource.Name != "JIRA_API_TOKEN" {
		t.Fatalf("secret source = %#v", resolved.SecretSource)
	}
}

func TestResolveProfileFlagIsCaseInsensitive(t *testing.T) {
	resolved := Resolve(Values{Profile: "Private"}, map[string]string{}, ProfileFile{
		Profiles: map[string]Profile{
			"Private": {Type: "server", BaseURL: "https://jira.example.com", User: "agent", Token: "profile-token"},
		},
	})
	if resolved.Profile != "private" {
		t.Fatalf("profile = %q", resolved.Profile)
	}
	if resolved.Secret != "profile-token" {
		t.Fatalf("secret = %q", resolved.Secret)
	}
	if got := Validate(resolved, map[string]string{}); len(got) != 0 {
		t.Fatalf("missing = %#v, want empty", got)
	}
}

func TestResolveTracksPasswordEnvSource(t *testing.T) {
	resolved := Resolve(Values{}, map[string]string{"JIRA_PASSWORD": "secret"}, ProfileFile{})
	if resolved.TokenEnv != "JIRA_PASSWORD" {
		t.Fatalf("token env = %q", resolved.TokenEnv)
	}
	if resolved.TokenEnv == "secret" {
		t.Fatal("resolved config stored secret value in TokenEnv")
	}
	if resolved.Secret != "secret" {
		t.Fatalf("secret = %q", resolved.Secret)
	}
	if resolved.SecretSource != (SecretSource{Kind: SecretSourceEnv, Name: "JIRA_PASSWORD"}) {
		t.Fatalf("secret source = %#v", resolved.SecretSource)
	}
}

func TestResolveProfileSecretFallbacks(t *testing.T) {
	tests := []struct {
		name       string
		profile    Profile
		wantSecret string
		wantSource SecretSource
	}{
		{
			name:       "token",
			profile:    Profile{Token: "profile-token"},
			wantSecret: "profile-token",
			wantSource: SecretSource{Kind: SecretSourceProfileToken},
		},
		{
			name:       "password",
			profile:    Profile{Password: "profile-password"},
			wantSecret: "profile-password",
			wantSource: SecretSource{Kind: SecretSourceProfilePassword},
		},
		{
			name:       "token_env",
			profile:    Profile{TokenEnv: "JIRA_PROFILE_TOKEN"},
			wantSecret: "profile-env-secret",
			wantSource: SecretSource{Kind: SecretSourceEnv, Name: "JIRA_PROFILE_TOKEN"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resolved := Resolve(Values{}, map[string]string{"JIRA_PROFILE_TOKEN": "profile-env-secret"}, ProfileFile{
				DefaultProfile: "private",
				Profiles:       map[string]Profile{"private": tt.profile},
			})
			if resolved.Secret != tt.wantSecret {
				t.Fatalf("secret = %q", resolved.Secret)
			}
			if resolved.SecretSource != tt.wantSource {
				t.Fatalf("source = %#v", resolved.SecretSource)
			}
		})
	}
}

func TestResolveSecretPrecedence(t *testing.T) {
	profiles := ProfileFile{
		DefaultProfile: "private",
		Profiles: map[string]Profile{
			"private": {TokenEnv: "JIRA_PROFILE_TOKEN", Token: "profile-token"},
		},
	}

	resolved := Resolve(Values{TokenEnv: "JIRA_FLAG_TOKEN"}, map[string]string{
		"JIRA_API_TOKEN":     "api-token",
		"JIRA_PASSWORD":      "password-token",
		"JIRA_PROFILE_TOKEN": "profile-env-token",
		"JIRA_FLAG_TOKEN":    "flag-token",
	}, profiles)
	if resolved.Secret != "flag-token" || resolved.TokenEnv != "JIRA_FLAG_TOKEN" {
		t.Fatalf("flag token did not win: %#v", resolved)
	}

	resolved = Resolve(Values{}, map[string]string{
		"JIRA_API_TOKEN": "api-token",
		"JIRA_PASSWORD":  "password-token",
	}, profiles)
	if resolved.Secret != "api-token" || resolved.TokenEnv != "JIRA_API_TOKEN" {
		t.Fatalf("api token did not win: %#v", resolved)
	}

	resolved = Resolve(Values{}, map[string]string{"JIRA_PASSWORD": "password-token"}, profiles)
	if resolved.Secret != "password-token" || resolved.TokenEnv != "JIRA_PASSWORD" {
		t.Fatalf("password token did not win: %#v", resolved)
	}
}

func TestResolveMissingFlagTokenEnvDoesNotFallBack(t *testing.T) {
	resolved := Resolve(Values{TokenEnv: "JIRA_FLAG_TOKEN"}, map[string]string{}, ProfileFile{
		DefaultProfile: "private",
		Profiles: map[string]Profile{
			"private": {Token: "profile-token"},
		},
	})
	if resolved.Secret != "" {
		t.Fatalf("secret = %q, want empty", resolved.Secret)
	}
	if got := Validate(resolved, map[string]string{}); len(got) != 3 || !contains(got, "token") {
		t.Fatalf("missing = %#v, want token and profile non-secret fields", got)
	}
}

func TestResolveNoSecretProfileCanUseEnv(t *testing.T) {
	profiles := ProfileFile{
		DefaultProfile: "private",
		Profiles: map[string]Profile{
			"private": {Type: "server", BaseURL: "https://jira.example.com", User: "agent"},
		},
	}
	resolved := Resolve(Values{}, map[string]string{"JIRA_API_TOKEN": "env-token"}, profiles)
	if got := Validate(resolved, map[string]string{"JIRA_API_TOKEN": "env-token"}); len(got) != 0 {
		t.Fatalf("missing = %#v, want empty", got)
	}
}

func TestResolveNoSecretProfileValidatesMissingToken(t *testing.T) {
	profiles := ProfileFile{
		DefaultProfile: "private",
		Profiles: map[string]Profile{
			"private": {Type: "server", BaseURL: "https://jira.example.com", User: "agent"},
		},
	}
	resolved := Resolve(Values{}, map[string]string{}, profiles)
	if got := Validate(resolved, map[string]string{}); len(got) != 1 || got[0] != "token" {
		t.Fatalf("missing = %#v, want [token]", got)
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

func TestValidateAcceptsResolvedSecret(t *testing.T) {
	values := Values{Type: "server", BaseURL: "https://jira.example.com", User: "agent", Secret: "secret"}
	if got := Validate(values, map[string]string{}); len(got) != 0 {
		t.Fatalf("Validate with resolved secret = %#v, want empty", got)
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

func TestLoadProfileConfigRejectsCaseVariantProfileDuplicates(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.toml")
	content := `
[profiles.Private]
token = "first"

[profiles.private]
token = "second"
`
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	_, err := LoadProfileConfig(path)
	if err == nil {
		t.Fatal("expected case-variant duplicate profile rejection")
	}
	if !strings.Contains(err.Error(), "conflicts") {
		t.Fatalf("err = %v, want conflict", err)
	}
}

func TestParseProfileConfigRejectsSecretSourceProblems(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name: "duplicate",
			input: `
[profiles.private]
token_env = "JIRA_TOKEN"
token = "secret"
`,
			want: "multiple secret sources",
		},
		{
			name: "empty token_env",
			input: `
[profiles.private]
token_env = ""
`,
			want: "must not be empty",
		},
		{
			name: "empty token",
			input: `
[profiles.private]
token = ""
`,
			want: "must not be empty",
		},
		{
			name: "empty password",
			input: `
[profiles.private]
password = ""
`,
			want: "must not be empty",
		},
		{
			name: "non-string token",
			input: `
[profiles.private]
token = 123
`,
			want: "must be a string",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ParseProfileConfig(tt.input)
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("err = %v, want contains %q", err, tt.want)
			}
		})
	}
}

func TestParseProfileConfigRejectsUnknownKeys(t *testing.T) {
	_, err := ParseProfileConfig(`
[profiles.private]
base_url = "https://jira.example.com"
unknown = "value"
`)
	if err == nil || !strings.Contains(err.Error(), "invalid keys") {
		t.Fatalf("err = %v, want invalid keys", err)
	}
}

func TestParseProfileConfigRejectsCaseVariantKeys(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name: "root key",
			input: `
Default_Profile = "private"
`,
			want: "root key",
		},
		{
			name: "profile key",
			input: `
[profiles.private]
Base_URL = "https://jira.example.com"
`,
			want: "invalid keys",
		},
		{
			name: "duplicate secret key",
			input: `
[profiles.private]
token = "first"
Token = "second"
`,
			want: "conflicts",
		},
		{
			name: "non-string root value",
			input: `
default_profile = 123
`,
			want: "must be a string",
		},
		{
			name: "profiles root not table",
			input: `
profiles = "not-a-table"
`,
			want: "must be a table",
		},
		{
			name: "non-string profile value",
			input: `
[profiles.private]
base_url = 123
`,
			want: "must be a string",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ParseProfileConfig(tt.input)
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("err = %v, want contains %q", err, tt.want)
			}
		})
	}
}

func TestParseProfileConfigRejectsDottedProfiles(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{
			name: "quoted dotted name",
			input: `
[profiles."private.team"]
base_url = "https://jira.example.com"
`,
		},
		{
			name: "nested table",
			input: `
[profiles.private.team]
base_url = "https://jira.example.com"
`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ParseProfileConfig(tt.input)
			if err == nil || !strings.Contains(err.Error(), "dotted") {
				t.Fatalf("err = %v, want dotted/nested rejection", err)
			}
		})
	}
}

func contains(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}
