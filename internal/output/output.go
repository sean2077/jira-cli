package output

import (
	"encoding/json"
	"fmt"
	"io"
)

type Mode int

const (
	Compact Mode = iota
	JSON
	Raw
)

func ModeFromOptions(jsonMode, rawMode bool) Mode {
	switch {
	case jsonMode:
		return JSON
	case rawMode:
		return Raw
	default:
		return Compact
	}
}

func WriteVersion(w io.Writer, mode Mode, version string) error {
	switch mode {
	case JSON:
		return WriteJSON(w, struct {
			OK      bool   `json:"ok"`
			Kind    string `json:"kind"`
			Version string `json:"version"`
		}{
			OK:      true,
			Kind:    "version",
			Version: version,
		})
	default:
		_, err := fmt.Fprintf(w, "jira %s\n", version)
		return err
	}
}

func WriteCompact(w io.Writer, lines ...string) error {
	for _, line := range lines {
		if _, err := fmt.Fprintln(w, line); err != nil {
			return err
		}
	}
	return nil
}

func WriteJSON(w io.Writer, value any) error {
	encoder := json.NewEncoder(w)
	encoder.SetEscapeHTML(false)
	return encoder.Encode(value)
}

func WriteRaw(w io.Writer, body []byte) error {
	_, err := w.Write(body)
	return err
}
