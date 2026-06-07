package jiraskill

import "embed"

const Name = "jira-cli"

// Files contains the installable skill payload bundled into release binaries.
//
//go:embed SKILL.md agents/openai.yaml
var Files embed.FS
