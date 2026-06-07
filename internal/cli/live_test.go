package cli

import (
	"bytes"
	"os"
	"strings"
	"testing"
)

func TestLiveJiraProbeIsOptInAndReadOnly(t *testing.T) {
	if os.Getenv("JIRA_LIVE_TEST") != "1" {
		t.Skip("set JIRA_LIVE_TEST=1 to run optional live Jira tests")
	}
	missing := missingLiveCredentials()
	if len(missing) > 0 {
		t.Skipf("live Jira credentials missing (%s); skipping without a live call", strings.Join(missing, ","))
	}

	var stdout, stderr bytes.Buffer
	code := Main([]string{"probe", "--timeout", "10s"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("live probe failed: code=%d stdout=%q stderr=%q; no artifacts are created by this read-only test", code, stdout.String(), stderr.String())
	}
}

func TestLiveJiraPersistentArtifactsRequireExplicitProject(t *testing.T) {
	if os.Getenv("JIRA_LIVE_TEST") != "1" {
		t.Skip("set JIRA_LIVE_TEST=1 to run optional live Jira tests")
	}
	if os.Getenv("JIRA_LIVE_PROJECT") == "" {
		t.Skip("live write tests require a dedicated JIRA_LIVE_PROJECT; no persistent artifacts created. Manual cleanup rule for future write tests: delete any JIRA_CLI_TEST_* issues if Jira permissions prevent automated cleanup.")
	}
	t.Skip("live write tests are intentionally not implemented in this milestone; future write tests must create only JIRA_CLI_TEST_* artifacts and print manual cleanup instructions on cleanup failure")
}

func missingLiveCredentials() []string {
	var missing []string
	for _, name := range []string{"JIRA_BASE_URL"} {
		if os.Getenv(name) == "" {
			missing = append(missing, name)
		}
	}
	if os.Getenv("JIRA_USER") == "" && os.Getenv("JIRA_USER_EMAIL") == "" {
		missing = append(missing, "JIRA_USER|JIRA_USER_EMAIL")
	}
	if os.Getenv("JIRA_API_TOKEN") == "" && os.Getenv("JIRA_PASSWORD") == "" {
		missing = append(missing, "JIRA_API_TOKEN|JIRA_PASSWORD")
	}
	return missing
}
