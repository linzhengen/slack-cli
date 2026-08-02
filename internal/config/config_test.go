package config

import "testing"

func TestSaveLoadRoundTrip(t *testing.T) {
	t.Setenv("SLACK_CLI_CONFIG_DIR", t.TempDir())

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if len(cfg.Profiles) != 0 {
		t.Fatalf("expected empty profiles on first load, got %v", cfg.Profiles)
	}

	cfg.SetProfile("work", Profile{Token: "xoxb-abc", Team: "acme"}, true)
	if err := cfg.Save(); err != nil {
		t.Fatalf("Save: %v", err)
	}

	reloaded, err := Load()
	if err != nil {
		t.Fatalf("reload: %v", err)
	}
	if reloaded.Active != "work" {
		t.Fatalf("expected active profile 'work', got %q", reloaded.Active)
	}
	prof, ok := reloaded.Get("")
	if !ok {
		t.Fatal("expected active profile to resolve")
	}
	if prof.Token != "xoxb-abc" || prof.Team != "acme" {
		t.Fatalf("unexpected profile: %+v", prof)
	}
}

func TestRemoveProfileClearsActive(t *testing.T) {
	t.Setenv("SLACK_CLI_CONFIG_DIR", t.TempDir())

	cfg, _ := Load()
	cfg.SetProfile("a", Profile{Token: "t1"}, true)
	cfg.RemoveProfile("a")

	if cfg.Active != "" {
		t.Fatalf("expected active to be cleared, got %q", cfg.Active)
	}
	if _, ok := cfg.Get(""); ok {
		t.Fatal("expected no active profile after removal")
	}
}

func TestGetNamedProfile(t *testing.T) {
	t.Setenv("SLACK_CLI_CONFIG_DIR", t.TempDir())

	cfg, _ := Load()
	cfg.SetProfile("a", Profile{Token: "t1"}, true)
	cfg.SetProfile("b", Profile{Token: "t2"}, false)

	if cfg.Active != "a" {
		t.Fatalf("expected 'a' to remain active, got %q", cfg.Active)
	}
	prof, ok := cfg.Get("b")
	if !ok || prof.Token != "t2" {
		t.Fatalf("expected profile b with token t2, got %+v ok=%v", prof, ok)
	}
}
