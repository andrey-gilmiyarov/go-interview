package lab

import (
	"context"
	"testing"
)

func TestConfigIgnoresExternalDatabase(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://evil/production")
	t.Setenv("PGHOST", "external")
	t.Setenv("POSTGRES_LAB_PORT", "45432")
	c, e := Config()
	if e != nil {
		t.Fatal(e)
	}
	if c.ConnConfig.Host != "127.0.0.1" || c.ConnConfig.Database != "postgres_lab" {
		t.Fatal("external database selected")
	}
}
func TestInvalidPort(t *testing.T) {
	for _, v := range []string{"0", "5432;rm", "65536", "-1"} {
		t.Setenv("POSTGRES_LAB_PORT", v)
		if _, e := Config(); e == nil {
			t.Fatalf("accepted %q", v)
		}
	}
}
func TestRetryInvalidAttempts(t *testing.T) {
	if Retry(context.Background(), 0, func(context.Context) error { return nil }) == nil {
		t.Fatal("accepted zero attempts")
	}
}
