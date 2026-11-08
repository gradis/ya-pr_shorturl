package config

import (
	"flag"
	"os"
	"testing"
)

func TestParseIgnoresDatabaseConnString(t *testing.T) {
	t.Setenv("DATABASE_CONN_STRING", "postgres://app:root@localhost:5432/shortener?sslmode=disable")
	t.Setenv("FILE_STORAGE_PATH", "/tmp/storage.json")

	cfg := Parse()

	if cfg.DatabaseDSN != "" {
		t.Fatalf("expected empty DatabaseDSN, got %q", cfg.DatabaseDSN)
	}

	if cfg.FileStoragePath != "/tmp/storage.json" {
		t.Fatalf("expected file storage path, got %q", cfg.FileStoragePath)
	}
}

func TestAuditConfig(t *testing.T) {
	for _, tc := range []struct {
		name                               string
		args                               []string
		fileEnv, urlEnv, wantFile, wantURL string
	}{
		{name: "disabled"},
		{name: "flags", args: []string{"--audit-file=events.jsonl", "--audit-url=https://example.com/audit"}, wantFile: "events.jsonl", wantURL: "https://example.com/audit"},
		{name: "environment overrides flags", args: []string{"--audit-file=flag.jsonl", "--audit-url=https://flag.example"}, fileEnv: "env.jsonl", urlEnv: "https://env.example", wantFile: "env.jsonl", wantURL: "https://env.example"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			oldFlags, oldArgs := flag.CommandLine, os.Args
			t.Cleanup(func() { flag.CommandLine = oldFlags; os.Args = oldArgs })
			flag.CommandLine = flag.NewFlagSet("test", flag.ContinueOnError)
			os.Args = append([]string{"test"}, tc.args...)
			t.Setenv("AUDIT_FILE", tc.fileEnv)
			t.Setenv("AUDIT_URL", tc.urlEnv)
			cfg := Parse()
			if cfg.AuditFile != tc.wantFile || cfg.AuditURL != tc.wantURL {
				t.Fatalf("audit config: %q %q", cfg.AuditFile, cfg.AuditURL)
			}
		})
	}
}
