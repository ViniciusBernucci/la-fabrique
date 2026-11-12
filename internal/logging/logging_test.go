package logging

import (
	"bytes"
	"strings"
	"testing"
)

func TestSensitiveAttributesAreRedacted(t *testing.T) {
	var buf bytes.Buffer
	log := New(&buf, "info", "api")
	log.Info("evento", "database_url", "postgres://u:pw@h/db", "session_token", "abc", "project_id", "p1")
	out := buf.String()
	if strings.Contains(out, "pw@h") || strings.Contains(out, `"abc"`) {
		t.Fatalf("segredo vazou: %s", out)
	}
	if !strings.Contains(out, `"project_id":"p1"`) || !strings.Contains(out, `"service":"api"`) {
		t.Fatalf("campos esperados ausentes: %s", out)
	}
}
