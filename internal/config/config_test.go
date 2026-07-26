package config

import (
	"maps"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func envFunc(m map[string]string) LookupEnv {
	return func(k string) (string, bool) {
		v, ok := m[k]
		return v, ok
	}
}

func strp(s string) *string { return &s }

func fixturePath(name string) string {
	return filepath.Join("testdata", name)
}

func TestResolveDefaults(t *testing.T) {
	// No flags, no env, no config at /home -> built-ins.
	cfg, _ := Resolve(Flags{}, envFunc(nil), "/home")
	want := Defaults()
	if diff := cmp.Diff(want, cfg); diff != "" {
		t.Fatalf("Resolve() mismatch (-want +got):\n%s", diff)
	}
}

func TestResolveConfigFile(t *testing.T) {
	env := map[string]string{"OMNICTX_CONFIG": fixturePath("config_full.yaml")}
	cfg, _ := Resolve(Flags{}, envFunc(env), "/home")
	if cfg.Icons != false {
		t.Errorf("icons = %v, want false (from file)", cfg.Icons)
	}
	if cfg.Separator != " | " {
		t.Errorf("separator = %q, want %q", cfg.Separator, " | ")
	}
	if diff := cmp.Diff([]string{SegmentKube, SegmentCloud}, cfg.Segments); diff != "" {
		t.Errorf("segments mismatch (-want +got):\n%s", diff)
	}
	// Partial colors map merges over defaults.
	if cfg.Colors[SegmentAzure] != "green" || cfg.Colors[SegmentKube] != "magenta" {
		t.Errorf("colors = %v, want azure=green kube=magenta", cfg.Colors)
	}
	if cfg.Colors[SegmentNamespace] != "dim" {
		t.Errorf("namespace color = %q, want default dim", cfg.Colors[SegmentNamespace])
	}
}

func TestResolveBrokenConfigFallsBack(t *testing.T) {
	env := map[string]string{"OMNICTX_CONFIG": fixturePath("config_broken.yaml")}
	cfg, debug := Resolve(Flags{}, envFunc(env), "/home")
	if diff := cmp.Diff(Defaults(), cfg); diff != "" {
		t.Fatalf("broken config should fall back to defaults (-want +got):\n%s", diff)
	}
	if len(debug) == 0 {
		t.Errorf("expected a debug note about the broken config")
	}
}

func TestResolveDisabledViaConfig(t *testing.T) {
	env := map[string]string{"OMNICTX_CONFIG": fixturePath("config_disabled.yaml")}
	cfg, _ := Resolve(Flags{}, envFunc(env), "/home")
	if cfg.Enabled {
		t.Fatalf("enabled = true, want false from config")
	}
}

func TestPrecedenceEnvOverFile(t *testing.T) {
	env := map[string]string{
		"OMNICTX_SEPARATOR": "ENV_SEP",
		"OMNICTX_ICONS":     "true",
		"OMNICTX_CONFIG":    fixturePath("config_full.yaml"),
	}
	// File says separator " | " & icons false; env overrides both.
	cfg, _ := Resolve(Flags{}, envFunc(env), "/home")
	if cfg.Separator != "ENV_SEP" {
		t.Errorf("separator = %q, want ENV_SEP (env wins over file)", cfg.Separator)
	}
	if cfg.Icons != true {
		t.Errorf("icons = %v, want true (env wins over file)", cfg.Icons)
	}
}

func TestShellFlagOverEnv(t *testing.T) {
	env := map[string]string{"OMNICTX_SHELL": "zsh"}
	cfg, _ := Resolve(Flags{Shell: strp("bash")}, envFunc(env), "/home")
	if cfg.Shell != ShellBash {
		t.Errorf("shell = %q, want bash (flag over env)", cfg.Shell)
	}
}

func TestEnvOverFile(t *testing.T) {
	env := map[string]string{
		"OMNICTX_CONFIG":   fixturePath("config_full.yaml"),
		"OMNICTX_SEGMENTS": "azure,namespace",
	}
	cfg, _ := Resolve(Flags{}, envFunc(env), "/home")
	if diff := cmp.Diff([]string{SegmentCloud, SegmentNamespace}, cfg.Segments); diff != "" {
		t.Fatalf("segments mismatch, env should win over file (-want +got):\n%s", diff)
	}
}

func TestSegmentAliasesAndDedup(t *testing.T) {
	// az and azure both alias the single cloud slot, so the duplicate is dropped.
	env := map[string]string{"OMNICTX_SEGMENTS": "az, k8s , ns, azure, bogus"}
	cfg, _ := Resolve(Flags{}, envFunc(env), "/home")
	if diff := cmp.Diff([]string{SegmentCloud, SegmentKube, SegmentNamespace}, cfg.Segments); diff != "" {
		t.Fatalf("segments mismatch, want aliases normalized and dups/unknown dropped (-want +got):\n%s", diff)
	}
}

func TestCloudSelectionAndNormalize(t *testing.T) {
	// Default is auto.
	if cfg, _ := Resolve(Flags{}, envFunc(nil), "/home"); cfg.Cloud != CloudAuto {
		t.Errorf("default cloud = %q, want auto", cfg.Cloud)
	}
	// Env overrides default.
	cfg, _ := Resolve(Flags{}, envFunc(map[string]string{"OMNICTX_CLOUD": "aws"}), "/home")
	if cfg.Cloud != "aws" {
		t.Errorf("cloud = %q, want aws (from env)", cfg.Cloud)
	}
	// Valid values pass through; "none" is allowed.
	for _, v := range []string{"azure", "aws", "gcp", "none", "auto"} {
		env := map[string]string{"OMNICTX_CLOUD": v}
		if cfg, _ := Resolve(Flags{}, envFunc(env), "/home"); cfg.Cloud != v {
			t.Errorf("cloud %q normalized to %q, want unchanged", v, cfg.Cloud)
		}
	}
	// Garbage falls back to auto.
	env := map[string]string{"OMNICTX_CLOUD": "nonsense"}
	if cfg, _ := Resolve(Flags{}, envFunc(env), "/home"); cfg.Cloud != CloudAuto {
		t.Errorf("invalid cloud normalized to %q, want auto", cfg.Cloud)
	}
}

func TestShellOnlyFromFlagOrEnvNotFile(t *testing.T) {
	// Shell is never a config key; default is none even with a full config file.
	env := map[string]string{"OMNICTX_CONFIG": fixturePath("config_full.yaml")}
	cfg, _ := Resolve(Flags{}, envFunc(env), "/home")
	if cfg.Shell != ShellNone {
		t.Errorf("shell = %q, want none by default", cfg.Shell)
	}
	cfg, _ = Resolve(Flags{Shell: strp("bash")}, envFunc(nil), "/home")
	if cfg.Shell != ShellBash {
		t.Errorf("shell = %q, want bash from flag", cfg.Shell)
	}
	cfg, _ = Resolve(Flags{}, envFunc(map[string]string{"OMNICTX_SHELL": "zsh"}), "/home")
	if cfg.Shell != ShellZsh {
		t.Errorf("shell = %q, want zsh from env", cfg.Shell)
	}
}

func TestKubeToggleResolution(t *testing.T) {
	kubeFile := filepath.Join(t.TempDir(), "config.yaml")
	if err := os.WriteFile(kubeFile, []byte("kube: false\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name string
		env  map[string]string
		want bool
	}{
		{"default is true", nil, true},
		{"file kube: false wins over default", map[string]string{"OMNICTX_CONFIG": kubeFile}, false},
		{"env off wins over default", map[string]string{"OMNICTX_KUBE": "off"}, false},
		{"env on overrides file false", map[string]string{"OMNICTX_CONFIG": kubeFile, "OMNICTX_KUBE": "on"}, true},
		{"invalid env is ignored, file wins", map[string]string{"OMNICTX_CONFIG": kubeFile, "OMNICTX_KUBE": "banana"}, false},
		{"invalid env is ignored, default wins", map[string]string{"OMNICTX_KUBE": "banana"}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg, _ := Resolve(Flags{}, envFunc(tt.env), "/home")
			if cfg.Kube != tt.want {
				t.Errorf("Kube = %v, want %v", cfg.Kube, tt.want)
			}
		})
	}
}

// All boolean env vars accept on/off (any case) on top of ParseBool forms.
func TestBoolEnvsAcceptOnOff(t *testing.T) {
	tests := []struct {
		name  string
		env   map[string]string
		check func(Config) bool
	}{
		{"OMNICTX_ENABLED=off", map[string]string{"OMNICTX_ENABLED": "off"}, func(c Config) bool { return !c.Enabled }},
		{"OMNICTX_ENABLED=ON", map[string]string{"OMNICTX_ENABLED": "ON"}, func(c Config) bool { return c.Enabled }},
		{"OMNICTX_ICONS=Off", map[string]string{"OMNICTX_ICONS": "Off"}, func(c Config) bool { return !c.Icons }},
		{"OMNICTX_ICONS=on", map[string]string{"OMNICTX_ICONS": "on"}, func(c Config) bool { return c.Icons }},
		{"OMNICTX_KUBE=OFF", map[string]string{"OMNICTX_KUBE": "OFF"}, func(c Config) bool { return !c.Kube }},
		{"ParseBool forms still work", map[string]string{"OMNICTX_KUBE": "false"}, func(c Config) bool { return !c.Kube }},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg, _ := Resolve(Flags{}, envFunc(tt.env), "/home")
			if !tt.check(cfg) {
				t.Errorf("%s not applied: %+v", tt.name, cfg)
			}
		})
	}
}

func TestInvalidBoolEnvLeavesDebugNote(t *testing.T) {
	_, debug := Resolve(Flags{}, envFunc(map[string]string{"OMNICTX_KUBE": "banana"}), "/home")
	found := false
	for _, d := range debug {
		if strings.Contains(d, "OMNICTX_KUBE") {
			found = true
		}
	}
	if !found {
		t.Errorf("expected a debug note about OMNICTX_KUBE, got %v", debug)
	}
}

func TestAliasesFromFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.yaml")
	yaml := "aliases:\n  azure:\n    prod: \"Azure subscription 1\"\n  gcp:\n    w: work\n"
	if err := os.WriteFile(path, []byte(yaml), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg, _ := Resolve(Flags{}, envFunc(map[string]string{"OMNICTX_CONFIG": path}), "/home")
	if got := cfg.Aliases["azure"]["prod"]; got != "Azure subscription 1" {
		t.Errorf("azure alias = %q, want Azure subscription 1", got)
	}
	if got := cfg.Aliases["gcp"]["w"]; got != "work" {
		t.Errorf("gcp alias = %q, want work", got)
	}

	// Absent key -> nil map, safe to index.
	cfg, _ = Resolve(Flags{}, envFunc(nil), "/home")
	if cfg.Aliases != nil {
		t.Errorf("Aliases = %v, want nil by default", cfg.Aliases)
	}
	if v := cfg.Aliases["gcp"]["w"]; v != "" {
		t.Errorf("indexing nil aliases should yield empty, got %q", v)
	}
}

// aws_profile / aws_region are config-file-only keys: no OMNICTX_* env
// override exists for them — AWS's native env vars (AWS_PROFILE, AWS_REGION)
// fill the session-override role and are honored by the hook's pin logic, not
// by the config merge.
func TestAWSKeysFromFile(t *testing.T) {
	tests := []struct {
		name        string
		yaml        string // "" = no config file
		env         map[string]string
		wantProfile string
		wantRegion  string
	}{
		{"defaults are empty", "", nil, "", ""},
		{"both keys from file", "aws_profile: digital-dev\naws_region: eu-central-1\n", nil, "digital-dev", "eu-central-1"},
		{"profile alone leaves region empty", "aws_profile: prod\n", nil, "prod", ""},
		{"region alone leaves profile empty", "aws_region: us-east-1\n", nil, "", "us-east-1"},
		{
			"AWS-native env vars do not leak into the merge",
			"aws_profile: prod\n",
			map[string]string{"AWS_PROFILE": "stage", "AWS_REGION": "us-west-2"},
			"prod", "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			env := map[string]string{}
			maps.Copy(env, tt.env)
			if tt.yaml != "" {
				path := filepath.Join(t.TempDir(), "config.yaml")
				if err := os.WriteFile(path, []byte(tt.yaml), 0o644); err != nil {
					t.Fatal(err)
				}
				env["OMNICTX_CONFIG"] = path
			}
			cfg, _ := Resolve(Flags{}, envFunc(env), "/home")
			if cfg.AWSProfile != tt.wantProfile {
				t.Errorf("AWSProfile = %q, want %q", cfg.AWSProfile, tt.wantProfile)
			}
			if cfg.AWSRegion != tt.wantRegion {
				t.Errorf("AWSRegion = %q, want %q", cfg.AWSRegion, tt.wantRegion)
			}
		})
	}
}
