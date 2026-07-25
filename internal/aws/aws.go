// Package aws reads the active AWS profile and region offline from ~/.aws/config
// and the environment, without the AWS SDK or any network call.
//
// account-id is intentionally out of scope (it requires STS / network). All
// failure modes are graceful: an empty Reading so the prompt is never broken.
package aws

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"omnictx/internal/cloud"
	"omnictx/internal/ini"
)

// LookupEnv mirrors os.LookupEnv (aliased from cloud for a single definition).
type LookupEnv = cloud.LookupEnv

// Provider implements cloud.Provider for AWS.
type Provider struct{}

// New returns the AWS provider.
func New() Provider { return Provider{} }

// Key identifies the provider.
func (Provider) Key() string { return "aws" }

// Label is the segment prefix: the AWS Nerd Font glyph, or the ASCII "aws:".
func (Provider) Label(icons bool) string {
	if icons {
		return cloud.IconAWS
	}
	return "aws:"
}

// Present reports whether there is any local AWS signal, used by auto-detection.
func (Provider) Present(lookup LookupEnv, home string) bool {
	return present(lookup, home)
}

// Read resolves "profile" (+ "/region" when known). OK is gated on Present so a
// pinned-but-unconfigured AWS shows nothing rather than a bare "default".
func (Provider) Read(lookup LookupEnv, home string) cloud.Reading {
	if !present(lookup, home) {
		return cloud.Reading{}
	}
	text := resolveProfile(lookup)
	if region := resolveRegion(lookup, home, resolveProfile(lookup)); region != "" {
		text += "/" + region
	}
	return cloud.Reading{Text: text, OK: text != ""}
}

func present(lookup LookupEnv, home string) bool {
	if envSet(lookup, "AWS_PROFILE") || envSet(lookup, "AWS_VAULT") ||
		envSet(lookup, "AWS_REGION") || envSet(lookup, "AWS_DEFAULT_REGION") {
		return true
	}
	return fileExists(configPath(lookup, home)) ||
		fileExists(filepath.Join(home, ".aws", "credentials"))
}

// resolveProfile: AWS_PROFILE > AWS_VAULT > "default".
func resolveProfile(lookup LookupEnv) string {
	if v, ok := lookup("AWS_PROFILE"); ok && v != "" {
		return v
	}
	if v, ok := lookup("AWS_VAULT"); ok && v != "" {
		return v
	}
	return "default"
}

// resolveRegion: AWS_REGION > AWS_DEFAULT_REGION > the profile's region in
// ~/.aws/config (EffectiveRegion with no persisted override).
func resolveRegion(lookup LookupEnv, home, profile string) string {
	return EffectiveRegion(lookup, home, profile, "")
}

// EffectiveRegion resolves the displayed region: AWS_REGION >
// AWS_DEFAULT_REGION > the persisted `aws_region:` override > the profile's
// region in ~/.aws/config. Empty when none of the layers answer.
func EffectiveRegion(lookup LookupEnv, home, profile, override string) string {
	if v, ok := lookup("AWS_REGION"); ok && v != "" {
		return v
	}
	if v, ok := lookup("AWS_DEFAULT_REGION"); ok && v != "" {
		return v
	}
	if override != "" {
		return override
	}
	f, ok := ini.ParseFile(configPath(lookup, home))
	if !ok {
		return ""
	}
	if v, ok := f.Get(sectionFor(profile), "region"); ok {
		return v
	}
	return ""
}

// EffectiveProfile resolves the active profile including the persisted
// `aws_profile:` override: AWS_PROFILE > AWS_VAULT > override > "default".
func EffectiveProfile(lookup LookupEnv, override string) string {
	if v, ok := lookup("AWS_PROFILE"); ok && v != "" {
		return v
	}
	if v, ok := lookup("AWS_VAULT"); ok && v != "" {
		return v
	}
	if override != "" {
		return override
	}
	return "default"
}

// regionRe is the offline shape check for region names (eu-central-1,
// us-gov-west-1, ...). Existence can only be verified online, which is out of
// scope — a well-formed typo is caught by the prompt showing the wrong region.
var regionRe = regexp.MustCompile(`^[a-z]{2}(-[a-z]+)+-\d+$`)

// ValidRegion reports whether r is shaped like an AWS region name.
func ValidRegion(r string) bool {
	return regionRe.MatchString(r)
}

// DirectiveUnset is the "unset the variable" encoding of the hook's env-sync
// lines (empty = leave the variable alone, anything else = the value to
// export). It is only ever emitted for hook-owned values.
const DirectiveUnset = "-"

// Directive computes one env-sync line of the per-prompt hook protocol for a
// variable (AWS_PROFILE or AWS_REGION). configured is the persisted global
// value ("" = omnictx does not manage the variable), current is the variable's
// value in the session env, marker is the matching __OMNICTX_* marker (what
// the hook last exported there), and vaultSet reports AWS_VAULT — an aws-vault
// session owns its env entirely.
//
// The pin table: a session whose current value differs from the marker was set
// by something else (manual export, direnv, aws-vault) and is never stomped; a
// value that is empty or hook-owned follows the global state, including the
// unset directive when the global state was cleared.
func Directive(configured, current, marker string, vaultSet bool) string {
	switch {
	case vaultSet:
		return ""
	case configured != "":
		if current == "" || current == marker {
			return configured
		}
		return "" // session manually pinned
	case current != "" && current == marker:
		return DirectiveUnset // we set it, global cleared
	default:
		return ""
	}
}

// Profile is one entry of `cloud aws list`: a profile name plus its region
// from ~/.aws/config (empty when the config does not set one).
type Profile struct {
	Name   string
	Region string
}

// Profiles lists locally configured profiles: the sections of ~/.aws/config
// (with the "profile " prefix stripped) followed by names that exist only in
// ~/.aws/credentials, deduplicated, in file order. Regions come from the
// config file only — credential values are never read. Missing or unparsable
// files degrade to an empty (or partial) list.
func Profiles(lookup LookupEnv, home string) []Profile {
	var profiles []Profile
	seen := map[string]bool{}

	add := func(name string) {
		if name == "" || seen[name] {
			return
		}
		seen[name] = true
		profiles = append(profiles, Profile{Name: name})
	}

	cfg := ini.File{}
	if data, err := os.ReadFile(configPath(lookup, home)); err == nil {
		cfg = ini.Parse(data)
		for _, section := range ini.Sections(data) {
			switch {
			case section == "default":
				add("default")
			case strings.HasPrefix(section, "profile "):
				add(strings.TrimSpace(strings.TrimPrefix(section, "profile ")))
			}
			// Other section kinds (e.g. [sso-session ...]) are not profiles.
		}
	}
	if data, err := os.ReadFile(filepath.Join(home, ".aws", "credentials")); err == nil {
		// Credentials sections are bare profile names; only names are used.
		for _, section := range ini.Sections(data) {
			add(section)
		}
	}

	for i := range profiles {
		if v, ok := cfg.Get(sectionFor(profiles[i].Name), "region"); ok {
			profiles[i].Region = v
		}
	}
	return profiles
}

// CurrentProfile exposes the active-profile resolution (AWS_PROFILE >
// AWS_VAULT > "default") for the list view's CURRENT marker.
func CurrentProfile(lookup LookupEnv) string {
	return resolveProfile(lookup)
}

// UnknownProfileError reports a switch target that matches no locally
// configured profile; Available carries the names for the error message.
type UnknownProfileError struct {
	Name      string
	Available []string
}

func (e *UnknownProfileError) Error() string {
	available := "(none found)"
	if len(e.Available) > 0 {
		available = strings.Join(e.Available, ", ")
	}
	return fmt.Sprintf("unknown AWS profile %q (available: %s)", e.Name, available)
}

// ValidateProfile checks that name is a locally configured profile (a section
// of ~/.aws/config or ~/.aws/credentials, as listed by Profiles). An unknown
// name yields *UnknownProfileError (a usage error); when neither source file
// can be read at all the plain error marks a broken environment instead.
func ValidateProfile(lookup LookupEnv, home, name string) error {
	_, cfgErr := os.ReadFile(configPath(lookup, home))
	_, credErr := os.ReadFile(filepath.Join(home, ".aws", "credentials"))
	if cfgErr != nil && credErr != nil {
		return fmt.Errorf("cannot read AWS profiles: %v; %v", cfgErr, credErr)
	}

	profiles := Profiles(lookup, home)
	names := make([]string, len(profiles))
	for i, p := range profiles {
		names[i] = p.Name
		if p.Name == name {
			return nil
		}
	}
	return &UnknownProfileError{Name: name, Available: names}
}

// sectionFor maps a profile to its ~/.aws/config section: the default profile is
// "[default]"; every other profile is "[profile NAME]".
func sectionFor(profile string) string {
	if profile == "default" {
		return "default"
	}
	return "profile " + profile
}

// configPath honors AWS_CONFIG_FILE, else ~/.aws/config.
func configPath(lookup LookupEnv, home string) string {
	if v, ok := lookup("AWS_CONFIG_FILE"); ok && v != "" {
		return v
	}
	return filepath.Join(home, ".aws", "config")
}

func envSet(lookup LookupEnv, key string) bool {
	v, ok := lookup(key)
	return ok && v != ""
}

func fileExists(p string) bool {
	info, err := os.Stat(p)
	return err == nil && !info.IsDir()
}
