package config

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
)

type Values struct {
	Profile  string
	Type     string
	BaseURL  string
	User     string
	TokenEnv string
}

type Profile struct {
	Type     string
	BaseURL  string
	User     string
	TokenEnv string
}

type ProfileFile struct {
	DefaultProfile string
	Profiles       map[string]Profile
}

func Resolve(flags Values, env map[string]string, profiles ProfileFile) Values {
	selectedProfile := flags.Profile
	if selectedProfile == "" {
		selectedProfile = profiles.DefaultProfile
	}

	resolved := Values{
		Profile: selectedProfile,
		Type:    "server",
	}
	if profile, ok := profiles.Profiles[selectedProfile]; ok {
		resolved.Type = firstNonEmpty(profile.Type, resolved.Type)
		resolved.BaseURL = profile.BaseURL
		resolved.User = profile.User
		resolved.TokenEnv = profile.TokenEnv
	}

	if v := env["JIRA_TYPE"]; v != "" {
		resolved.Type = v
	}
	if v := env["JIRA_BASE_URL"]; v != "" {
		resolved.BaseURL = v
	}
	if v := firstNonEmpty(env["JIRA_USER_EMAIL"], env["JIRA_USER"]); v != "" {
		resolved.User = v
	}
	if env["JIRA_API_TOKEN"] != "" {
		resolved.TokenEnv = "JIRA_API_TOKEN"
	} else if env["JIRA_PASSWORD"] != "" {
		resolved.TokenEnv = "JIRA_PASSWORD"
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
	if flags.TokenEnv != "" {
		resolved.TokenEnv = flags.TokenEnv
	}

	return resolved
}

func ParseProfileConfig(input string) (ProfileFile, error) {
	file := ProfileFile{Profiles: map[string]Profile{}}
	currentProfile := ""
	scanner := bufio.NewScanner(strings.NewReader(input))

	for lineNumber := 1; scanner.Scan(); lineNumber++ {
		line := strings.TrimSpace(stripInlineComment(scanner.Text()))
		if line == "" {
			continue
		}
		if strings.HasPrefix(line, "[") && strings.HasSuffix(line, "]") {
			section := strings.TrimSpace(strings.TrimSuffix(strings.TrimPrefix(line, "["), "]"))
			if !strings.HasPrefix(section, "profiles.") || section == "profiles." {
				return file, fmt.Errorf("line %d: unsupported section %q", lineNumber, section)
			}
			currentProfile = strings.TrimPrefix(section, "profiles.")
			if _, ok := file.Profiles[currentProfile]; !ok {
				file.Profiles[currentProfile] = Profile{}
			}
			continue
		}

		key, rawValue, ok := strings.Cut(line, "=")
		if !ok {
			return file, fmt.Errorf("line %d: expected key = value", lineNumber)
		}
		key = strings.TrimSpace(key)
		value, err := parseScalar(strings.TrimSpace(rawValue))
		if err != nil {
			return file, fmt.Errorf("line %d: %w", lineNumber, err)
		}
		if isSecretKey(key) {
			return file, fmt.Errorf("line %d: plaintext secret key %q is not allowed", lineNumber, key)
		}

		if currentProfile == "" {
			if key != "default_profile" {
				return file, fmt.Errorf("line %d: unsupported root key %q", lineNumber, key)
			}
			file.DefaultProfile = value
			continue
		}

		profile := file.Profiles[currentProfile]
		switch key {
		case "type":
			profile.Type = value
		case "base_url":
			profile.BaseURL = value
		case "user":
			profile.User = value
		case "token_env":
			profile.TokenEnv = value
		default:
			return file, fmt.Errorf("line %d: unsupported profile key %q", lineNumber, key)
		}
		file.Profiles[currentProfile] = profile
	}
	if err := scanner.Err(); err != nil {
		return file, err
	}
	return file, nil
}

func LoadProfileConfig(path string) (ProfileFile, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return ProfileFile{Profiles: map[string]Profile{}}, nil
		}
		return ProfileFile{}, err
	}
	return ParseProfileConfig(string(content))
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
	if values.TokenEnv == "" {
		missing = append(missing, "token-env")
	} else if env[values.TokenEnv] == "" {
		missing = append(missing, "token")
	}
	return missing
}

func parseScalar(raw string) (string, error) {
	if raw == "" {
		return "", errors.New("empty values are not supported")
	}
	if strings.HasPrefix(raw, "\"") {
		value, err := strconv.Unquote(raw)
		if err != nil {
			return "", err
		}
		return value, nil
	}
	return raw, nil
}

func stripInlineComment(line string) string {
	inQuote := false
	escaped := false
	for i, r := range line {
		if escaped {
			escaped = false
			continue
		}
		if r == '\\' && inQuote {
			escaped = true
			continue
		}
		if r == '"' {
			inQuote = !inQuote
			continue
		}
		if r == '#' && !inQuote {
			return line[:i]
		}
	}
	return line
}

func isSecretKey(key string) bool {
	switch strings.ToLower(key) {
	case "token", "password", "api_token", "jira_api_token", "jira_password":
		return true
	default:
		return false
	}
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}
