package aws

import (
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"omnictx/internal/cloud"
)

func env(m map[string]string) LookupEnv {
	return func(k string) (string, bool) {
		v, ok := m[k]
		return v, ok
	}
}

func fixture(name string) string { return filepath.Join("..", "..", "testdata", name) }

func TestReadDisplayAndProfilePrecedence(t *testing.T) {
	def := fixture("aws_config_default.ini")
	named := fixture("aws_config_named.ini")

	cases := []struct {
		name string
		env  map[string]string
		want string
	}{
		{"AWS_PROFILE wins over AWS_VAULT", map[string]string{"AWS_CONFIG_FILE": def, "AWS_PROFILE": "foo", "AWS_VAULT": "bar"}, "foo"},
		{"AWS_VAULT when no AWS_PROFILE", map[string]string{"AWS_CONFIG_FILE": def, "AWS_VAULT": "bar"}, "bar"},
		{"default profile + region from config", map[string]string{"AWS_CONFIG_FILE": def}, "default/us-east-1"},
		{"named profile region from [profile NAME]", map[string]string{"AWS_CONFIG_FILE": named, "AWS_PROFILE": "prod"}, "prod/eu-west-1"},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			got := New().Read(env(tt.env), "/nonexistent")
			if !got.OK || got.Text != tt.want {
				t.Fatalf("Read() = %q/%v, want %q", got.Text, got.OK, tt.want)
			}
		})
	}
}

// credentialsHome builds a temp home dir with the given file as
// ~/.aws/credentials (fixture name resolved via fixture()).
func credentialsHome(t *testing.T, name string) string {
	t.Helper()
	home := t.TempDir()
	if err := os.MkdirAll(filepath.Join(home, ".aws"), 0o755); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(fixture(name))
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(home, ".aws", "credentials"), data, 0o600); err != nil {
		t.Fatal(err)
	}
	return home
}

func TestValidateProfile(t *testing.T) {
	named := fixture("aws_config_named.ini")

	t.Run("known profile from config", func(t *testing.T) {
		if err := ValidateProfile(env(map[string]string{"AWS_CONFIG_FILE": named}), "/nonexistent", "prod"); err != nil {
			t.Fatalf("ValidateProfile(prod) = %v, want nil", err)
		}
	})

	t.Run("profile only in credentials", func(t *testing.T) {
		home := credentialsHome(t, "aws_credentials_extra.ini")
		lookup := env(map[string]string{"AWS_CONFIG_FILE": filepath.Join(home, "missing")})
		if err := ValidateProfile(lookup, home, "ci-only"); err != nil {
			t.Fatalf("ValidateProfile(ci-only) = %v, want nil", err)
		}
	})

	t.Run("unknown profile is a typed usage error", func(t *testing.T) {
		err := ValidateProfile(env(map[string]string{"AWS_CONFIG_FILE": named}), "/nonexistent", "nope")
		var unknown *UnknownProfileError
		if !errors.As(err, &unknown) {
			t.Fatalf("ValidateProfile(nope) = %v, want *UnknownProfileError", err)
		}
		if !reflect.DeepEqual(unknown.Available, []string{"default", "prod"}) {
			t.Errorf("Available = %v, want [default prod]", unknown.Available)
		}
	})

	t.Run("unreadable sources are a plain error", func(t *testing.T) {
		home := t.TempDir() // no ~/.aws at all
		lookup := env(map[string]string{"AWS_CONFIG_FILE": filepath.Join(home, "missing")})
		err := ValidateProfile(lookup, home, "anything")
		if err == nil {
			t.Fatal("ValidateProfile with no readable sources should fail")
		}
		var unknown *UnknownProfileError
		if errors.As(err, &unknown) {
			t.Fatalf("broken environment must not be a usage error, got %v", err)
		}
	})

	t.Run("broken config still validates via credentials", func(t *testing.T) {
		home := credentialsHome(t, "aws_credentials_extra.ini")
		lookup := env(map[string]string{"AWS_CONFIG_FILE": fixture("aws_config_broken.ini")})
		if err := ValidateProfile(lookup, home, "ci-only"); err != nil {
			t.Fatalf("ValidateProfile(ci-only) = %v, want nil", err)
		}
	})
}

func TestReadRegionPrecedence(t *testing.T) {
	def := fixture("aws_config_default.ini")
	noRegion := fixture("aws_config_no_region.ini")

	cases := []struct {
		name string
		env  map[string]string
		want string
	}{
		{"AWS_REGION wins", map[string]string{"AWS_CONFIG_FILE": def, "AWS_REGION": "r1", "AWS_DEFAULT_REGION": "r2"}, "default/r1"},
		{"AWS_DEFAULT_REGION next", map[string]string{"AWS_CONFIG_FILE": def, "AWS_DEFAULT_REGION": "r2"}, "default/r2"},
		{"config region last", map[string]string{"AWS_CONFIG_FILE": def}, "default/us-east-1"},
		{"no region anywhere -> profile only", map[string]string{"AWS_CONFIG_FILE": noRegion}, "default"},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			if got := New().Read(env(tt.env), "/nonexistent"); got.Text != tt.want {
				t.Fatalf("Read() = %q, want %q", got.Text, tt.want)
			}
		})
	}
}

func TestValidRegion(t *testing.T) {
	valid := []string{"eu-central-1", "us-east-1", "us-gov-west-1", "ap-southeast-3", "cn-north-1"}
	for _, r := range valid {
		if !ValidRegion(r) {
			t.Errorf("ValidRegion(%q) = false, want true", r)
		}
	}
	invalid := []string{"", "Frankfurt", "eu-central", "eu-central-", "eu-central-1a", "EU-CENTRAL-1", "eu_central_1", "e-central-1", "eu-central-x", " eu-central-1"}
	for _, r := range invalid {
		if ValidRegion(r) {
			t.Errorf("ValidRegion(%q) = true, want false", r)
		}
	}
}

func TestEffectiveRegion(t *testing.T) {
	def := fixture("aws_config_default.ini")

	cases := []struct {
		name     string
		env      map[string]string
		override string
		want     string
	}{
		{"AWS_REGION wins over everything", map[string]string{"AWS_CONFIG_FILE": def, "AWS_REGION": "r1", "AWS_DEFAULT_REGION": "r2"}, "eu-central-1", "r1"},
		{"AWS_DEFAULT_REGION beats the override", map[string]string{"AWS_CONFIG_FILE": def, "AWS_DEFAULT_REGION": "r2"}, "eu-central-1", "r2"},
		{"override beats the profile config", map[string]string{"AWS_CONFIG_FILE": def}, "eu-central-1", "eu-central-1"},
		{"no override falls back to profile config", map[string]string{"AWS_CONFIG_FILE": def}, "", "us-east-1"},
		{"nothing anywhere is empty", map[string]string{"AWS_CONFIG_FILE": fixture("aws_config_no_region.ini")}, "", ""},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			if got := EffectiveRegion(env(tt.env), "/nonexistent", "default", tt.override); got != tt.want {
				t.Fatalf("EffectiveRegion() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestEffectiveProfile(t *testing.T) {
	cases := []struct {
		name     string
		env      map[string]string
		override string
		want     string
	}{
		{"AWS_PROFILE wins", map[string]string{"AWS_PROFILE": "p", "AWS_VAULT": "v"}, "cfg", "p"},
		{"AWS_VAULT next", map[string]string{"AWS_VAULT": "v"}, "cfg", "v"},
		{"override beats the built-in default", nil, "cfg", "cfg"},
		{"nothing set falls back to default", nil, "", "default"},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			if got := EffectiveProfile(env(tt.env), tt.override); got != tt.want {
				t.Fatalf("EffectiveProfile() = %q, want %q", got, tt.want)
			}
		})
	}
}

// TestDirective walks the full pin table: aws-vault owns the session; a
// configured value follows into empty or hook-owned sessions but never stomps
// a manual pin; a cleared global state unsets only hook-owned values.
func TestDirective(t *testing.T) {
	cases := []struct {
		name       string
		configured string
		current    string
		marker     string
		vault      bool
		want       string
	}{
		{"AWS_VAULT owns the session even with config", "prod", "dev", "dev", true, ""},
		{"AWS_VAULT owns the session even without config", "", "prod", "prod", true, ""},
		{"configured + empty env exports", "prod", "", "", false, "prod"},
		{"configured + hook-owned env follows the switch", "prod", "dev", "dev", false, "prod"},
		{"configured + hook-owned env re-exports same value", "prod", "prod", "prod", false, "prod"},
		{"configured + manual pin is untouched", "prod", "stage", "prod", false, ""},
		{"configured + manual pin without marker is untouched", "prod", "stage", "", false, ""},
		{"absent + hook-owned env unsets", "", "prod", "prod", false, DirectiveUnset},
		{"absent + manual value is untouched", "", "stage", "", false, ""},
		{"absent + stale marker without value does nothing", "", "", "prod", false, ""},
		{"absent + empty env does nothing", "", "", "", false, ""},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			if got := Directive(tt.configured, tt.current, tt.marker, tt.vault); got != tt.want {
				t.Fatalf("Directive(%q, %q, %q, %v) = %q, want %q",
					tt.configured, tt.current, tt.marker, tt.vault, got, tt.want)
			}
		})
	}
}

func TestReadMissingIsEmpty(t *testing.T) {
	// No env signal and no ~/.aws files -> nothing to show.
	got := New().Read(env(nil), t.TempDir())
	if got.OK || got.Text != "" {
		t.Fatalf("Read() = %q/%v, want empty/false", got.Text, got.OK)
	}
}

func TestReadBrokenConfigDegrades(t *testing.T) {
	// A broken config file still exists (present), but yields no region.
	got := New().Read(env(map[string]string{"AWS_CONFIG_FILE": fixture("aws_config_broken.ini")}), "/nonexistent")
	if got.Text != "default" {
		t.Fatalf("Read() = %q, want %q (broken config -> profile only)", got.Text, "default")
	}
}

func TestPresent(t *testing.T) {
	if New().Present(env(nil), t.TempDir()) {
		t.Error("no files and no env should not be present")
	}
	if !New().Present(env(map[string]string{"AWS_CONFIG_FILE": fixture("aws_config_default.ini")}), "/nonexistent") {
		t.Error("existing config file should be present")
	}
	if !New().Present(env(map[string]string{"AWS_REGION": "us-east-1"}), t.TempDir()) {
		t.Error("AWS_REGION env should make it present")
	}
}

func TestKeyAndLabel(t *testing.T) {
	p := New()
	if p.Key() != "aws" {
		t.Errorf("Key() = %q, want aws", p.Key())
	}
	if p.Label(true) != cloud.IconAWS {
		t.Errorf("Label(icons) = %q, want %q", p.Label(true), cloud.IconAWS)
	}
	if p.Label(false) != "aws:" {
		t.Errorf("Label(ascii) = %q, want aws:", p.Label(false))
	}
}

func TestProfiles(t *testing.T) {
	t.Run("config sections with regions, prefix stripped", func(t *testing.T) {
		got := Profiles(env(map[string]string{"AWS_CONFIG_FILE": fixture("aws_config_named.ini")}), t.TempDir())
		want := []Profile{{Name: "default", Region: "us-east-1"}, {Name: "prod", Region: "eu-west-1"}}
		if !reflect.DeepEqual(got, want) {
			t.Errorf("Profiles() = %v, want %v", got, want)
		}
	})

	t.Run("credentials-only profile is appended without secrets", func(t *testing.T) {
		home := t.TempDir()
		if err := os.MkdirAll(filepath.Join(home, ".aws"), 0o755); err != nil {
			t.Fatal(err)
		}
		src, _ := os.ReadFile(fixture("aws_credentials_extra.ini"))
		if err := os.WriteFile(filepath.Join(home, ".aws", "credentials"), src, 0o600); err != nil {
			t.Fatal(err)
		}
		got := Profiles(env(map[string]string{"AWS_CONFIG_FILE": fixture("aws_config_named.ini")}), home)
		want := []Profile{
			{Name: "default", Region: "us-east-1"},
			{Name: "prod", Region: "eu-west-1"},
			{Name: "ci-only"}, // from credentials; no region, no key material
		}
		if !reflect.DeepEqual(got, want) {
			t.Errorf("Profiles() = %v, want %v", got, want)
		}
	})

	t.Run("missing files yield nothing", func(t *testing.T) {
		if got := Profiles(env(map[string]string{"AWS_CONFIG_FILE": fixture("nope.ini")}), t.TempDir()); got != nil {
			t.Errorf("Profiles() = %v, want nil", got)
		}
	})

	t.Run("broken config still yields credentials names", func(t *testing.T) {
		home := t.TempDir()
		if err := os.MkdirAll(filepath.Join(home, ".aws"), 0o755); err != nil {
			t.Fatal(err)
		}
		src, _ := os.ReadFile(fixture("aws_credentials_extra.ini"))
		if err := os.WriteFile(filepath.Join(home, ".aws", "credentials"), src, 0o600); err != nil {
			t.Fatal(err)
		}
		got := Profiles(env(map[string]string{"AWS_CONFIG_FILE": fixture("aws_config_broken.ini")}), home)
		for _, p := range got {
			if p.Name == "ci-only" {
				return
			}
		}
		t.Errorf("Profiles() = %v, want ci-only present", got)
	})
}

func TestCurrentProfile(t *testing.T) {
	if got := CurrentProfile(env(nil)); got != "default" {
		t.Errorf("CurrentProfile() = %q, want default", got)
	}
	if got := CurrentProfile(env(map[string]string{"AWS_PROFILE": "prod"})); got != "prod" {
		t.Errorf("CurrentProfile() = %q, want prod", got)
	}
}
