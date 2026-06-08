package config

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/pelletier/go-toml/v2"
	"github.com/spf13/viper"
)

type Values struct {
	Profile      string
	Type         string
	BaseURL      string
	User         string
	TokenEnv     string
	Secret       string
	SecretSource SecretSource
	AuthScheme   string
}

type SecretSource struct {
	Kind string
	Name string
}

type Profile struct {
	Type     string `mapstructure:"type"`
	BaseURL  string `mapstructure:"base_url"`
	User     string `mapstructure:"user"`
	TokenEnv string `mapstructure:"token_env"`
	Token    string `mapstructure:"token"`
	Password string `mapstructure:"password"`
	Auth     string `mapstructure:"auth"`
}

type ProfileFile struct {
	DefaultProfile string             `mapstructure:"default_profile"`
	Profiles       map[string]Profile `mapstructure:"profiles"`
}

const (
	SecretSourceEnv             = "env"
	SecretSourceProfileToken    = "profile_token"
	SecretSourceProfilePassword = "profile_password"
)

func Resolve(flags Values, env map[string]string, profiles ProfileFile) Values {
	selectedProfile := normalizeProfileName(flags.Profile)
	if selectedProfile == "" {
		selectedProfile = normalizeProfileName(profiles.DefaultProfile)
	}

	resolved := Values{
		Profile: selectedProfile,
		Type:    "server",
	}
	profile, hasProfile := lookupProfile(profiles.Profiles, selectedProfile)
	if hasProfile {
		resolved.Type = FirstNonEmpty(profile.Type, resolved.Type)
		resolved.BaseURL = profile.BaseURL
		resolved.User = profile.User
	}

	if v := env["JIRA_TYPE"]; v != "" {
		resolved.Type = v
	}
	if v := env["JIRA_BASE_URL"]; v != "" {
		resolved.BaseURL = v
	}
	if v := FirstNonEmpty(env["JIRA_USER_EMAIL"], env["JIRA_USER"]); v != "" {
		resolved.User = v
	}

	if flags.Type != "" {
		resolved.Type = flags.Type
	}
	if flags.BaseURL != "" {
		resolved.BaseURL = flags.BaseURL
	}
	if flags.User != "" {
		resolved.User = flags.User
	}

	if hasProfile {
		resolved.AuthScheme = profile.Auth
	}
	if v := env["JIRA_AUTH_SCHEME"]; v != "" {
		resolved.AuthScheme = v
	}
	if flags.AuthScheme != "" {
		resolved.AuthScheme = flags.AuthScheme
	}
	resolved.AuthScheme = strings.ToLower(strings.TrimSpace(resolved.AuthScheme))

	if flags.TokenEnv != "" {
		resolved.TokenEnv = flags.TokenEnv
		resolved.Secret = env[flags.TokenEnv]
		if resolved.Secret != "" {
			resolved.SecretSource = SecretSource{Kind: SecretSourceEnv, Name: flags.TokenEnv}
		}
	} else if env["JIRA_API_TOKEN"] != "" {
		resolved.TokenEnv = "JIRA_API_TOKEN"
		resolved.Secret = env["JIRA_API_TOKEN"]
		resolved.SecretSource = SecretSource{Kind: SecretSourceEnv, Name: "JIRA_API_TOKEN"}
	} else if env["JIRA_PASSWORD"] != "" {
		resolved.TokenEnv = "JIRA_PASSWORD"
		resolved.Secret = env["JIRA_PASSWORD"]
		resolved.SecretSource = SecretSource{Kind: SecretSourceEnv, Name: "JIRA_PASSWORD"}
	} else if hasProfile && profile.TokenEnv != "" {
		resolved.TokenEnv = profile.TokenEnv
		resolved.Secret = env[profile.TokenEnv]
		if resolved.Secret != "" {
			resolved.SecretSource = SecretSource{Kind: SecretSourceEnv, Name: profile.TokenEnv}
		}
	} else if hasProfile && profile.Token != "" {
		resolved.Secret = profile.Token
		resolved.SecretSource = SecretSource{Kind: SecretSourceProfileToken}
	} else if hasProfile && profile.Password != "" {
		resolved.Secret = profile.Password
		resolved.SecretSource = SecretSource{Kind: SecretSourceProfilePassword}
	}

	return resolved
}

func ParseProfileConfig(input string) (ProfileFile, error) {
	if err := validateRawProfileTables([]byte(input)); err != nil {
		return emptyProfileFile(), err
	}
	v := newProfileViper()
	if err := v.ReadConfig(strings.NewReader(input)); err != nil {
		return emptyProfileFile(), err
	}
	return decodeProfileConfig(v)
}

func LoadProfileConfig(path string) (ProfileFile, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return emptyProfileFile(), nil
		}
		return ProfileFile{}, err
	}
	if err := validateRawProfileTables(raw); err != nil {
		return emptyProfileFile(), err
	}
	v := newProfileViper()
	if err := v.ReadConfig(bytes.NewReader(raw)); err != nil {
		var notFound viper.ConfigFileNotFoundError
		if errors.As(err, &notFound) {
			return emptyProfileFile(), nil
		}
		return ProfileFile{}, err
	}
	return decodeProfileConfig(v)
}

func EnvMap(environ []string) map[string]string {
	env := make(map[string]string, len(environ))
	for _, item := range environ {
		key, value, ok := strings.Cut(item, "=")
		if !ok {
			continue
		}
		env[key] = value
	}
	return env
}

func Validate(values Values, env map[string]string) []string {
	var missing []string
	if values.Type == "" {
		missing = append(missing, "type")
	} else if values.Type != "server" {
		missing = append(missing, "type=server")
	}
	if values.BaseURL == "" {
		missing = append(missing, "base-url")
	}
	if values.User == "" {
		missing = append(missing, "user")
	}
	if values.Secret == "" && (values.TokenEnv == "" || env[values.TokenEnv] == "") {
		missing = append(missing, "token")
	}
	return missing
}

func newProfileViper() *viper.Viper {
	v := viper.New()
	v.SetConfigType("toml")
	return v
}

func decodeProfileConfig(v *viper.Viper) (ProfileFile, error) {
	var file ProfileFile
	if err := v.UnmarshalExact(&file); err != nil {
		return emptyProfileFile(), err
	}
	if file.Profiles == nil {
		file.Profiles = map[string]Profile{}
	}
	if err := normalizeProfileFile(&file); err != nil {
		return emptyProfileFile(), err
	}
	return file, nil
}

func validateRawProfileTables(input []byte) error {
	var raw map[string]any
	if err := toml.Unmarshal(input, &raw); err != nil {
		return err
	}
	if err := validateRawRootKeys(raw); err != nil {
		return err
	}
	rawProfiles, hasProfiles := raw["profiles"]
	if !hasProfiles || rawProfiles == nil {
		return nil
	}
	profiles, ok := asStringMap(rawProfiles)
	if !ok {
		return fmt.Errorf("root key %q must be a table", "profiles")
	}
	seen := map[string]string{}
	for profileName, rawProfile := range profiles {
		if strings.Contains(profileName, ".") {
			return fmt.Errorf("profile %q uses dotted profile name; dotted profile names are not supported", profileName)
		}
		normalizedName := normalizeProfileName(profileName)
		if normalizedName == "" {
			return fmt.Errorf("profile name must not be empty")
		}
		if previous, exists := seen[normalizedName]; exists && previous != profileName {
			return fmt.Errorf("profile %q conflicts with profile %q after case-insensitive normalization", profileName, previous)
		}
		seen[normalizedName] = profileName
		profileSettings, ok := asStringMap(rawProfile)
		if !ok {
			return fmt.Errorf("profile %q must be a table", profileName)
		}
		if err := validateRawProfileKeys(profileName, profileSettings); err != nil {
			return err
		}
	}
	return nil
}

func validateRawRootKeys(settings map[string]any) error {
	seen := map[string]string{}
	for key := range settings {
		normalizedKey := strings.ToLower(key)
		if previous, exists := seen[normalizedKey]; exists && previous != key {
			return fmt.Errorf("root key %q conflicts with root key %q after case-insensitive normalization", key, previous)
		}
		seen[normalizedKey] = key
	}
	for key := range settings {
		if _, known := allowedRootKeys[key]; !known {
			return fmt.Errorf("root key %q is not supported; config keys are case-sensitive", key)
		}
		if key == "default_profile" {
			if _, ok := settings[key].(string); !ok {
				return fmt.Errorf("root key %q must be a string", key)
			}
		}
	}
	return nil
}

func validateRawProfileKeys(profileName string, settings map[string]any) error {
	seen := map[string]string{}
	for key := range settings {
		normalizedKey := strings.ToLower(key)
		if previous, exists := seen[normalizedKey]; exists && previous != key {
			return fmt.Errorf("profile %q key %q conflicts with key %q after case-insensitive normalization", profileName, key, previous)
		}
		seen[normalizedKey] = key
	}
	for key, value := range settings {
		if _, known := allowedProfileKeys[key]; known {
			continue
		}
		if nested, ok := asStringMap(value); ok && nested != nil {
			return fmt.Errorf("profile %q contains nested key %q; dotted/nested profile tables are not supported", profileName, key)
		}
		return fmt.Errorf("profile %q contains invalid keys: %s; config keys are case-sensitive", profileName, key)
	}
	var secretKeys []string
	for _, key := range []string{"token_env", "token", "password"} {
		if value, ok := settings[key]; ok {
			text, ok := value.(string)
			if !ok {
				return fmt.Errorf("profile %q key %q must be a string", profileName, key)
			}
			if text == "" {
				return fmt.Errorf("profile %q key %q must not be empty", profileName, key)
			}
			secretKeys = append(secretKeys, key)
		}
	}
	if len(secretKeys) > 1 {
		return fmt.Errorf("profile %q configures multiple secret sources: %s", profileName, strings.Join(secretKeys, ","))
	}
	for key, value := range settings {
		if _, ok := value.(string); !ok {
			return fmt.Errorf("profile %q key %q must be a string", profileName, key)
		}
	}
	return nil
}

var allowedRootKeys = map[string]struct{}{
	"default_profile": {},
	"profiles":        {},
}

var allowedProfileKeys = map[string]struct{}{
	"type":      {},
	"base_url":  {},
	"user":      {},
	"token_env": {},
	"token":     {},
	"password":  {},
	"auth":      {},
}

func asStringMap(value any) (map[string]any, bool) {
	if value == nil {
		return nil, false
	}
	if typed, ok := value.(map[string]any); ok {
		return typed, true
	}
	return nil, false
}

func emptyProfileFile() ProfileFile {
	return ProfileFile{Profiles: map[string]Profile{}}
}

func normalizeProfileFile(file *ProfileFile) error {
	file.DefaultProfile = normalizeProfileName(file.DefaultProfile)
	normalized := make(map[string]Profile, len(file.Profiles))
	for name, profile := range file.Profiles {
		normalizedName := normalizeProfileName(name)
		if _, exists := normalized[normalizedName]; exists {
			return fmt.Errorf("profile %q conflicts after case-insensitive normalization", name)
		}
		normalized[normalizedName] = profile
	}
	file.Profiles = normalized
	return nil
}

func normalizeProfileName(name string) string {
	return strings.ToLower(strings.TrimSpace(name))
}

func lookupProfile(profiles map[string]Profile, name string) (Profile, bool) {
	if profiles == nil {
		return Profile{}, false
	}
	if profile, ok := profiles[name]; ok {
		return profile, true
	}
	for profileName, profile := range profiles {
		if normalizeProfileName(profileName) == name {
			return profile, true
		}
	}
	return Profile{}, false
}

// FirstNonEmpty returns the first non-empty string in values, or "" if all are
// empty. It is the single shared implementation aliased by other packages.
func FirstNonEmpty(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}
