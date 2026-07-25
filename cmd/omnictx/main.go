// Command omnictx prints a prompt segment showing the active Azure
// subscription, the current kube-context, and its namespace.
//
// Core invariant: it NEVER breaks the prompt. Any error in normal (render) mode
// results in empty/partial output and exit 0. A top-level recover guards
// against panics.
package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"text/tabwriter"

	"omnictx/internal/aws"
	"omnictx/internal/azure"
	"omnictx/internal/cloud"
	"omnictx/internal/config"
	"omnictx/internal/gcp"
	"omnictx/internal/kube"
	"omnictx/internal/render"
	"omnictx/internal/shellinit"
)

// Version is overridable at build time via -ldflags "-X main.Version=...".
var Version = "dev"

func main() {
	// The render path must never break the prompt: swallow any panic and exit 0.
	defer func() {
		if r := recover(); r != nil {
			os.Exit(0)
		}
	}()

	args := os.Args[1:]
	if len(args) > 0 {
		switch args[0] {
		case "init":
			os.Exit(runInit(args[1:]))
		case "on":
			os.Exit(runEnable(true))
		case "off":
			os.Exit(runEnable(false))
		case "cloud":
			os.Exit(runCloud(args[1:], os.Stdout, os.Stderr))
		case "kube":
			os.Exit(runKube(args[1:], os.Stdout, os.Stderr))
		case "ns", "namespace":
			os.Exit(runNamespace(args[1:], os.Stdout, os.Stderr))
		case "hook":
			runHook(args[1:], os.Stdout)
			os.Exit(0)
		}
	}

	runRender(args)
	os.Exit(0)
}

// runInit handles `omnictx init <bash|zsh>`. This is not prompt-render mode, so
// usage errors return a non-zero code to surface mistakes during setup.
func runInit(args []string) int {
	if len(args) < 1 {
		fmt.Fprintln(os.Stderr, "usage: omnictx init <bash|zsh>")
		return 2
	}
	shell := args[0]
	cmd := selfCommand()
	code, err := shellinit.Generate(shell, cmd)
	if err != nil {
		fmt.Fprintln(os.Stderr, "omnictx:", err)
		return 2
	}
	if isTTY(os.Stdout) {
		fmt.Fprintf(os.Stderr, "Hint: add this line to ~/.%src:\n  eval \"$(%s init %s)\"\n", shell, cmd, shell)
		return 0
	}
	fmt.Print(code)
	return 0
}

func isTTY(f *os.File) bool {
	fi, err := f.Stat()
	return err == nil && (fi.Mode()&os.ModeCharDevice) != 0
}

// selfCommand returns the command used inside the generated shell snippet.
// We use the bare binary name so the snippet relies on PATH (the documented
// install path), matching starship/zoxide/direnv conventions.
func selfCommand() string {
	return "omnictx"
}

func runRender(args []string) {
	flags, showVersion, showHelp, ok := parseRenderArgs(args)
	if !ok {
		// Never break the prompt over a bad flag.
		return
	}

	if showHelp {
		printUsage(os.Stdout)
		return
	}

	if showVersion {
		fmt.Printf("omnictx %s\n", Version)
		return
	}

	home, _ := os.UserHomeDir()
	cfg, _ := config.Resolve(flags, os.LookupEnv, home)

	if !cfg.Enabled {
		// Disabled (omnictx off / OMNICTX_ENABLED=false): print nothing, exit 0.
		return
	}

	data := gather(cfg, home, os.LookupEnv)
	out := render.Render(data, cfg)
	fmt.Print(out)
}

// runHook implements the per-prompt hook invocation (`omnictx hook --shell
// <bash|zsh>`, emitted by `init` snippets): exactly three newline-terminated
// lines — the AWS_PROFILE directive, the AWS_REGION directive, and the prompt
// segment. Directive encoding: empty = leave the variable alone, "-" = unset
// it, anything else = export it. This is render mode with two extra lines, so
// the core invariant applies in full: never write anything, degrade any error
// to empty directives and a best-effort segment, always exit 0. The segment is
// rendered as if the directives were already applied, so the prompt is correct
// on the same cycle that applies a switch. The enabled mute silences ONLY the
// segment line: directives keep flowing, because the mute controls display,
// not state — matching gcp/azure/kube, whose switches take effect under the
// mute too (their tools read the switched files directly).
func runHook(args []string, stdout io.Writer) {
	flags, _, _, ok := parseRenderArgs(args)
	if !ok {
		_, _ = fmt.Fprint(stdout, "\n\n\n")
		return
	}
	home, _ := os.UserHomeDir()
	cfg, _ := config.Resolve(flags, os.LookupEnv, home)

	envValue := func(k string) string { v, _ := os.LookupEnv(k); return v }
	vault := envValue("AWS_VAULT") != ""
	profileDir := aws.Directive(cfg.AWSProfile, envValue("AWS_PROFILE"), envValue("__OMNICTX_AWS_PROFILE"), vault)
	regionDir := aws.Directive(cfg.AWSRegion, envValue("AWS_REGION"), envValue("__OMNICTX_AWS_REGION"), vault)

	out := ""
	if cfg.Enabled {
		lookup := directiveLookup(os.LookupEnv, map[string]string{
			"AWS_PROFILE": profileDir,
			"AWS_REGION":  regionDir,
		})
		out = render.Render(gather(cfg, home, lookup), cfg)
	}
	_, _ = fmt.Fprintf(stdout, "%s\n%s\n%s\n", profileDir, regionDir, out)
}

// directiveLookup overlays pending hook directives on the real environment so
// the segment reflects the exports/unsets that ship in the same output: an
// export directive reads as the variable's value, an unset directive hides it,
// an empty directive falls through to the session env.
func directiveLookup(base config.LookupEnv, directives map[string]string) config.LookupEnv {
	return func(k string) (string, bool) {
		if d, ok := directives[k]; ok && d != "" {
			if d == aws.DirectiveUnset {
				return "", false
			}
			return d, true
		}
		return base(k)
	}
}

// printUsage writes the grouped, human-readable help.
func printUsage(w io.Writer) {
	_, _ = fmt.Fprintln(w, `omnictx — print a shell-prompt segment with your active cloud
provider (Azure/AWS/GCP), current kube-context, and namespace.

Usage:
  omnictx [--shell <bash|zsh|none>]   print the segment (used by the prompt hook)
  omnictx init <bash|zsh>             print shell integration code
  eval "$(omnictx init bash)"         typical install (add to ~/.bashrc)

Subcommands:
  init <bash|zsh>   shell integration (add to ~/.bashrc / ~/.zshrc)
  on / off          persist enabled: true/false to config file (affects all future shells)
  cloud [azure|aws|gcp|auto|none|on|off]
                    persist the active cloud to config file; off hides the slot,
                    on returns to auto-detect; without an argument prints the
                    effective value (OMNICTX_CLOUD overrides per-session)
  cloud [azure|aws|gcp] list
                    offline table of that provider's local accounts (AWS profiles,
                    gcloud configurations, Azure subscriptions); bare "cloud list"
                    uses the active provider
  cloud <azure|aws|gcp> <account>
                    switch the active account: azure flips isDefault in
                    azureProfile.json (name or id), gcp activates the named
                    configuration, aws persists aws_profile in omnictx's own
                    config — shells running the omnictx hook export it as
                    AWS_PROFILE on their next prompt (~/.aws is never written;
                    a manual export AWS_PROFILE=<name> still pins its session);
                    accepts short aliases from the config file and pins that
                    provider as the displayed cloud on success
  cloud aws region [<region>|auto]
                    persist an aws_region override in the omnictx config, on
                    top of the active profile's own region; "auto" clears the
                    override, no argument prints the effective region
  kube [<context>|list|on|off]
                    switch the current kube-context (rewrites current-context in
                    kubeconfig); no argument prints the current one, "list" shows
                    all available contexts; on/off toggle the kube segment in the
                    config file and never touch kubeconfig (OMNICTX_KUBE overrides
                    per-session)
  ns [<name>|list]  (alias: namespace)
                    switch the namespace of the active kube-context (rewrites
                    that context entry in kubeconfig); no argument prints the
                    current namespace. "list" queries the cluster via kubectl
                    (needs kubectl on PATH) — the only online subcommand; the
                    prompt render itself always stays offline

Flags:
  --version                   print version and exit
  -h, --help                  print this help and exit`)
}

// parseRenderArgs parses CLI args into config.Flags. ok=false signals a parse
// error (the caller then exits silently to protect the prompt).
func parseRenderArgs(args []string) (flags config.Flags, showVersion, showHelp, ok bool) {
	fs := flag.NewFlagSet("omnictx", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	fs.Usage = func() {}

	shell := fs.String("shell", "", "color escaping mode: bash|zsh|none")
	version := fs.Bool("version", false, "print version and exit")

	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return config.Flags{}, false, true, true
		}
		return config.Flags{}, false, false, false
	}

	fs.Visit(func(f *flag.Flag) {
		if f.Name == "shell" {
			flags.Shell = shell
		}
	})

	return flags, *version, false, true
}

// globalConfigPath returns the path to the config file honoring OMNICTX_CONFIG.
func globalConfigPath() string {
	if p := os.Getenv("OMNICTX_CONFIG"); p != "" {
		return p
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".config", "omnictx", "config.yaml")
}

// setConfigKeys updates (or creates) the config file, changing only the
// lines for the given key/value pairs and preserving all other content
// including comments. One read, one write, regardless of how many keys.
func setConfigKeys(path string, pairs ...string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		if !os.IsNotExist(err) {
			return err
		}
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			return err
		}
	}

	lines := strings.Split(string(data), "\n")
	for i := 0; i+1 < len(pairs); i += 2 {
		key, value := pairs[i], pairs[i+1]
		newLine := key + ": " + value
		prefix := key + ":"
		replaced := false
		for j, l := range lines {
			// Column-0 prefix only: an indented "key:" belongs to a nested
			// block (e.g. colors.kube), and replacing it would corrupt the YAML.
			if strings.HasPrefix(l, prefix) {
				lines[j] = newLine
				replaced = true
				break
			}
		}
		if !replaced {
			lines = append([]string{newLine}, lines...)
		}
	}
	return os.WriteFile(path, []byte(strings.Join(lines, "\n")), 0o644)
}

// removeConfigKey deletes the top-level line for the given key from the config
// file, preserving everything else (other keys, comments, nested blocks — the
// same column-0 discipline as setConfigKeys). Removal is idempotent: a missing
// file or an absent key is success, and the file is never created.
func removeConfigKey(path, key string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}

	prefix := key + ":"
	lines := strings.Split(string(data), "\n")
	kept := lines[:0]
	removed := false
	for _, l := range lines {
		if !removed && strings.HasPrefix(l, prefix) {
			removed = true
			continue
		}
		kept = append(kept, l)
	}
	if !removed {
		return nil
	}
	return os.WriteFile(path, []byte(strings.Join(kept, "\n")), 0o644)
}

// runEnable handles `omnictx on` and `omnictx off`. `on` means "show
// everything": besides lifting the mute it turns hidden parts back on —
// kube: false becomes true, cloud: none becomes auto (a concrete provider
// pin is already visible and stays untouched). `off` only sets the mute.
func runEnable(enabled bool) int {
	path := globalConfigPath()
	pairs := []string{"enabled", "false"}
	if enabled {
		pairs = []string{"enabled", "true"}
		t := config.ReadFileToggles(path)
		if !t.Kube {
			pairs = append(pairs, "kube", "true")
		}
		if t.Cloud == config.CloudNone {
			pairs = append(pairs, "cloud", config.CloudAuto)
		}
	}
	if err := setConfigKeys(path, pairs...); err != nil {
		fmt.Fprintf(os.Stderr, "omnictx: %v\n", err)
		return 1
	}
	return 0
}

const cloudUsage = "usage: omnictx cloud [azure|aws|gcp|auto|none|on|off]\n" +
	"       omnictx cloud [azure|aws|gcp] list\n" +
	"       omnictx cloud <azure|aws|gcp> <account>\n" +
	"       omnictx cloud aws region [<region>|auto]"

// runCloud handles `omnictx cloud [value]` and the read-only listing forms.
// With no argument it prints the effective selection (env > config > default).
// `list` (reserved, never persisted) prints the active provider's accounts;
// `<provider> list` prints that provider's accounts. With one value argument
// it persists the value to the config file, like `on`/`off` do for enabled.
// This is interactive setup mode, so unlike render's normalizeCloud (which
// silently falls back to auto to protect the prompt) an unknown value is
// rejected loudly.
func runCloud(args []string, stdout, stderr io.Writer) int {
	home, _ := os.UserHomeDir()

	if len(args) == 0 {
		cfg, notes := config.Resolve(config.Flags{}, os.LookupEnv, home)
		warnAll(stderr, notes)
		_, _ = fmt.Fprintln(stdout, cfg.Cloud)
		return 0
	}
	// `region` is a reserved word (like `list`): `cloud aws region ...`
	// dispatches to the region subcommand and can never name a profile.
	if len(args) >= 2 && strings.ToLower(strings.TrimSpace(args[0])) == "aws" &&
		strings.ToLower(strings.TrimSpace(args[1])) == "region" {
		return runAwsRegion(args[2:], home, stdout, stderr)
	}
	if len(args) > 2 {
		_, _ = fmt.Fprintln(stderr, cloudUsage)
		return 2
	}

	if len(args) == 2 {
		provider := strings.ToLower(strings.TrimSpace(args[0]))
		form := strings.ToLower(strings.TrimSpace(args[1]))
		isProvider := provider == "azure" || provider == "aws" || provider == "gcp"
		switch {
		case form == "list":
			if !isProvider {
				_, _ = fmt.Fprintln(stderr, cloudUsage)
				return 2
			}
			if provider == "azure" {
				warnAll(stderr, azure.Check(os.LookupEnv, home))
			}
			printCloudList(stdout, provider, home)
			return 0
		case isProvider:
			// Two-argument switch form: the second word is the account
			// (`use` is no longer a reserved keyword).
			return runCloudSwitch(provider, args[1], home, stderr)
		default:
			// A non-provider first argument (auto/none/on/off/unknown) never
			// takes a second argument.
			_, _ = fmt.Fprintln(stderr, cloudUsage)
			return 2
		}
	}

	if strings.ToLower(strings.TrimSpace(args[0])) == "list" {
		// Bare `cloud list`: the effective provider, selected exactly like
		// render does; none selected -> quiet, exit 0.
		cfg, notes := config.Resolve(config.Flags{}, os.LookupEnv, home)
		warnAll(stderr, notes)
		if active, ok := cloud.Select(cloudProviders(), cfg.Cloud, os.LookupEnv, home); ok {
			if active.Key() == "azure" {
				warnAll(stderr, azure.Check(os.LookupEnv, home))
			}
			printCloudList(stdout, active.Key(), home)
		}
		return 0
	}

	v := strings.ToLower(strings.TrimSpace(args[0]))
	// on/off are display-toggle aliases: off hides the slot, on returns to
	// auto-detect (a previous provider pin is not remembered). The literal
	// `on` while globally muted (omnictx off) lifts the mute but must expose
	// only the part being turned on: the kube segment stays hidden
	// (kube: false) until its own `kube on`. Plain `cloud auto` and `off`
	// never touch the mute.
	isOn := false
	switch v {
	case "on":
		v = config.CloudAuto
		isOn = true
	case "off":
		v = config.CloudNone
	}
	switch v {
	case "azure", "aws", "gcp", config.CloudAuto, config.CloudNone:
	default:
		_, _ = fmt.Fprintf(stderr, "omnictx: invalid cloud %q\n%s\n", args[0], cloudUsage)
		return 2
	}

	pairs := []string{"cloud", v}
	if isOn && !config.ReadFileToggles(globalConfigPath()).Enabled {
		pairs = append(pairs, "kube", "false", "enabled", "true")
	}
	if err := setConfigKeys(globalConfigPath(), pairs...); err != nil {
		_, _ = fmt.Fprintf(stderr, "omnictx: %v\n", err)
		return 1
	}
	return 0
}

const kubeUsage = "usage: omnictx kube [<context>|list|on|off]"

// runKube handles `omnictx kube [<context>|list|on|off]`. No argument prints
// the current context; the reserved words (contexts with those literal names
// are not switchable here) come first: `list` prints all contexts with the
// current one marked, `on`/`off` persist the kube display toggle to omnictx's
// own config and never touch a kubeconfig (`on` also re-enables omnictx
// globally, see below). Any other argument validates
// against the parsed kubeconfigs and then rewrites current-context via
// kube.WriteContext. That switch is the only code path in omnictx that writes
// to a file it does not own — render mode never does.
func runKube(args []string, stdout, stderr io.Writer) int {
	home, _ := os.UserHomeDir()

	if len(args) == 0 {
		if ctx := kube.Read(os.LookupEnv, home).Context; ctx != "" {
			_, _ = fmt.Fprintln(stdout, ctx)
		}
		return 0
	}
	if len(args) > 1 {
		_, _ = fmt.Fprintln(stderr, kubeUsage)
		return 2
	}

	switch args[0] {
	case "on", "off":
		val := "true"
		if args[0] == "off" {
			val = "false"
		}
		// `kube on` while globally muted lifts the mute but must expose only
		// the part being turned on: the cloud slot stays hidden (cloud: none)
		// until its own `cloud on`. `kube off` never touches the mute.
		pairs := []string{"kube", val}
		if args[0] == "on" && !config.ReadFileToggles(globalConfigPath()).Enabled {
			pairs = append(pairs, "cloud", config.CloudNone, "enabled", "true")
		}
		if err := setConfigKeys(globalConfigPath(), pairs...); err != nil {
			_, _ = fmt.Fprintf(stderr, "omnictx: %v\n", err)
			return 1
		}
		return 0
	}

	if args[0] == "list" {
		warnAll(stderr, kube.Check(os.LookupEnv, home))
		printKubeTable(stdout, kube.Contexts(os.LookupEnv, home), kube.Read(os.LookupEnv, home).Context)
		return 0
	}

	target := args[0]
	entries := kube.Contexts(os.LookupEnv, home)
	names := make([]string, len(entries))
	found := false
	for i, e := range entries {
		names[i] = e.Name
		if e.Name == target {
			found = true
		}
	}
	if !found {
		available := "(none found)"
		if len(names) > 0 {
			available = strings.Join(names, ", ")
		}
		_, _ = fmt.Fprintf(stderr, "omnictx: unknown context %q\navailable contexts: %s\n%s\n", target, available, kubeUsage)
		return 2
	}

	if err := kube.WriteContext(os.LookupEnv, home, target); err != nil {
		_, _ = fmt.Fprintf(stderr, "omnictx: %v\n", err)
		return 1
	}
	return 0
}

const namespaceUsage = "usage: omnictx ns [<name>|list]"

// runNamespace handles `omnictx ns [<name>|list]` (alias: `namespace`). No
// argument prints the active context's namespace (empty prints nothing). One
// argument switches the namespace of the active kube-context by rewriting its
// entry in the kubeconfig — the second omnictx path that writes to a file it
// does not own, and only on an explicit user command. The name is validated as
// a DNS-1123 label first (invalid → exit 2, no write); kubeconfig-state
// problems (no active context, context not defined, unlocatable/broken) fail
// loudly with exit 1. `list` is a subcommand word, never a switch target: it
// queries the cluster via kubectl (see runNamespaceList) — render mode stays
// strictly offline and shares no code with that path.
func runNamespace(args []string, stdout, stderr io.Writer) int {
	home, _ := os.UserHomeDir()

	if len(args) == 0 {
		if ns := kube.Read(os.LookupEnv, home).Namespace; ns != "" {
			_, _ = fmt.Fprintln(stdout, ns)
		}
		return 0
	}
	if len(args) > 1 {
		_, _ = fmt.Fprintln(stderr, namespaceUsage)
		return 2
	}

	name := args[0]
	if name == "list" {
		return runNamespaceList(stdout, stderr, home)
	}
	if !kube.ValidNamespace(name) {
		_, _ = fmt.Fprintf(stderr, "omnictx: invalid namespace %q (must be a DNS-1123 label)\n%s\n", name, namespaceUsage)
		return 2
	}
	if err := kube.WriteNamespace(os.LookupEnv, home, name); err != nil {
		_, _ = fmt.Fprintf(stderr, "omnictx: %v\n", err)
		return 1
	}
	return 0
}

// runNamespaceList handles `omnictx ns list` — the ONLY online code path in
// the binary: the namespace list lives in the cluster, so it shells out to
// kubectl (time-bounded by --request-timeout so a dead VPN fails in seconds).
// Render mode never reaches this code and stays strictly offline. The result
// is a CURRENT/NAME table marking the active context's namespace as resolved
// by the offline kube.Read (`default` when the context sets none). Failures
// are environment problems, not usage errors: kubectl missing from PATH or
// exiting non-zero → its stderr passes through with an omnictx line, exit 1.
// This path never writes any file.
func runNamespaceList(stdout, stderr io.Writer, home string) int {
	cmd := exec.Command("kubectl", "get", "namespaces", "-o", "name", "--request-timeout=10s")
	var out strings.Builder
	cmd.Stdout = &out
	cmd.Stderr = stderr
	if err := cmd.Run(); err != nil {
		if errors.Is(err, exec.ErrNotFound) {
			_, _ = fmt.Fprintln(stderr, "omnictx: ns list needs kubectl on PATH to query the cluster, and kubectl was not found")
			return 1
		}
		_, _ = fmt.Fprintf(stderr, "omnictx: kubectl get namespaces failed: %v\n", err)
		return 1
	}

	current := kube.Read(os.LookupEnv, home).Namespace
	if current == "" {
		// Kubernetes' effective default when the context sets no namespace.
		current = "default"
	}

	var rows [][]string
	for line := range strings.SplitSeq(out.String(), "\n") {
		// `-o name` prints one `namespace/<name>` per line; skip anything else.
		name := strings.TrimPrefix(strings.TrimSpace(line), "namespace/")
		if name == "" || name == strings.TrimSpace(line) {
			continue
		}
		marker := ""
		if name == current {
			marker = "*"
		}
		rows = append(rows, []string{marker, name})
	}
	printTable(stdout, []string{"CURRENT", "NAME"}, rows)
	return 0
}

// runCloudSwitch handles `omnictx cloud <provider> <account>`: switching the
// provider's active account. For gcloud and Azure that state lives in a local
// file (active_config, azureProfile.json isDefault). AWS has no such file, so
// the switch persists `aws_profile:` to omnictx's own config — the per-prompt
// hook exports it as AWS_PROFILE in every hook-running shell; ~/.aws is never
// written. The account argument goes through the `aliases` config key first;
// names/ids are otherwise matched verbatim.
func runCloudSwitch(provider, account string, home string, stderr io.Writer) int {
	account = strings.TrimSpace(account)

	cfg, notes := config.Resolve(config.Flags{}, os.LookupEnv, home)
	warnAll(stderr, notes)
	if canon := cfg.Aliases[provider][account]; canon != "" {
		account = canon
	}

	switch provider {
	case "gcp":
		if err := gcp.Use(os.LookupEnv, home, account); err != nil {
			code := 1
			var unknown *gcp.UnknownConfigError
			if errors.As(err, &unknown) {
				code = 2
			}
			_, _ = fmt.Fprintf(stderr, "omnictx: %v\n", err)
			return code
		}
		return pinCloudAfterUse(provider, stderr)
	case "azure":
		if err := azure.Use(os.LookupEnv, home, account); err != nil {
			code := 1
			var unknown *azure.UnknownAccountError
			var ambiguous *azure.AmbiguousAccountError
			if errors.As(err, &unknown) || errors.As(err, &ambiguous) {
				code = 2
			}
			_, _ = fmt.Fprintf(stderr, "omnictx: %v\n", err)
			return code
		}
		return pinCloudAfterUse(provider, stderr)
	case "aws":
		if err := aws.ValidateProfile(os.LookupEnv, home, account); err != nil {
			code := 1
			var unknown *aws.UnknownProfileError
			if errors.As(err, &unknown) {
				code = 2
			}
			_, _ = fmt.Fprintf(stderr, "omnictx: %v\n", err)
			return code
		}
		if err := setConfigKeys(globalConfigPath(), "aws_profile", account); err != nil {
			_, _ = fmt.Fprintf(stderr, "omnictx: %v\n", err)
			return 1
		}
		if code := pinCloudAfterUse(provider, stderr); code != 0 {
			return code
		}
		_, _ = fmt.Fprintf(stderr,
			"omnictx: AWS profile switched to %q; shells running the omnictx hook apply it on their next prompt\n", account)
		return 0
	default:
		_, _ = fmt.Fprintln(stderr, cloudUsage)
		return 2
	}
}

const awsRegionUsage = "usage: omnictx cloud aws region [<region>|auto]"

// runAwsRegion handles `omnictx cloud aws region [<region>|auto]`. One
// argument persists an `aws_region:` override to omnictx's own config
// (validated offline by shape only — existence would need network); `auto`
// removes the override idempotently so the region falls back to the active
// profile's configured region. No argument prints the effective region
// (env > override > profile config), read-only.
func runAwsRegion(args []string, home string, stdout, stderr io.Writer) int {
	if len(args) > 1 {
		_, _ = fmt.Fprintln(stderr, awsRegionUsage)
		return 2
	}

	if len(args) == 0 {
		cfg, notes := config.Resolve(config.Flags{}, os.LookupEnv, home)
		warnAll(stderr, notes)
		profile := aws.EffectiveProfile(os.LookupEnv, cfg.AWSProfile)
		if r := aws.EffectiveRegion(os.LookupEnv, home, profile, cfg.AWSRegion); r != "" {
			_, _ = fmt.Fprintln(stdout, r)
		}
		return 0
	}

	v := strings.TrimSpace(args[0])
	if strings.ToLower(v) == "auto" {
		if err := removeConfigKey(globalConfigPath(), "aws_region"); err != nil {
			_, _ = fmt.Fprintf(stderr, "omnictx: %v\n", err)
			return 1
		}
		return 0
	}
	if !aws.ValidRegion(v) {
		_, _ = fmt.Fprintf(stderr, "omnictx: invalid region %q\n%s\n", args[0], awsRegionUsage)
		return 2
	}
	if err := setConfigKeys(globalConfigPath(), "aws_region", v); err != nil {
		_, _ = fmt.Fprintf(stderr, "omnictx: %v\n", err)
		return 1
	}
	return 0
}

// warnAll prints interactive-mode warnings to stderr. Render never calls it:
// the prompt stays silent about broken sources by design, but a human typing
// a subcommand deserves to know why their config is being ignored.
func warnAll(stderr io.Writer, notes []string) {
	for _, n := range notes {
		_, _ = fmt.Fprintf(stderr, "omnictx: warning: %s\n", n)
	}
}

// pinCloudAfterUse persists `cloud: <provider>` after a successful account
// switch, so the prompt immediately shows the provider that was just switched
// to (instead of whatever the previous pin/auto-detection displayed). It
// deliberately never touches the display toggles: a switch changes state, not
// visibility — under a persisted mute (enabled: false) the state still flips,
// and the prompt reflects it once `on` / `cloud on` lifts the mute.
func pinCloudAfterUse(provider string, stderr io.Writer) int {
	if err := setConfigKeys(globalConfigPath(), "cloud", provider); err != nil {
		_, _ = fmt.Fprintf(stderr, "omnictx: account switched, but pinning the cloud failed: %v\n", err)
		return 1
	}
	return 0
}

// printTable renders a kubectl-style table with tabwriter alignment. The
// header appears only when there is at least one row, so empty listings stay
// quiet (no output, exit 0).
func printTable(stdout io.Writer, header []string, rows [][]string) {
	if len(rows) == 0 {
		return
	}
	w := tabwriter.NewWriter(stdout, 0, 0, 3, ' ', 0)
	_, _ = fmt.Fprintln(w, strings.Join(header, "\t"))
	for _, r := range rows {
		_, _ = fmt.Fprintln(w, strings.Join(r, "\t"))
	}
	_ = w.Flush()
}

// printKubeTable renders `kube list` as a kubectl-get-contexts-style table.
func printKubeTable(stdout io.Writer, entries []kube.ContextEntry, current string) {
	rows := make([][]string, 0, len(entries))
	for _, e := range entries {
		marker := ""
		if e.Name == current {
			marker = "*"
		}
		rows = append(rows, []string{marker, e.Name, e.Cluster, e.AuthInfo, e.Namespace})
	}
	printTable(stdout, []string{"CURRENT", "NAME", "CLUSTER", "AUTHINFO", "NAMESPACE"}, rows)
}

// printCloudList renders `cloud <provider> list`: the provider's locally
// configured accounts, read from the same offline sources as render.
func printCloudList(stdout io.Writer, key, home string) {
	switch key {
	case "aws":
		current := aws.CurrentProfile(os.LookupEnv)
		var rows [][]string
		for _, p := range aws.Profiles(os.LookupEnv, home) {
			marker := ""
			if p.Name == current {
				marker = "*"
			}
			rows = append(rows, []string{marker, p.Name, p.Region})
		}
		printTable(stdout, []string{"CURRENT", "NAME", "REGION"}, rows)
	case "gcp":
		current := gcp.CurrentConfiguration(os.LookupEnv, home)
		var rows [][]string
		for _, c := range gcp.Configurations(os.LookupEnv, home) {
			marker := ""
			if c.Name == current {
				marker = "*"
			}
			rows = append(rows, []string{marker, c.Name, c.Account, c.Project})
		}
		printTable(stdout, []string{"CURRENT", "NAME", "ACCOUNT", "PROJECT"}, rows)
	case "azure":
		var rows [][]string
		for _, s := range azure.Subscriptions(os.LookupEnv, home) {
			marker := ""
			if s.IsDefault {
				marker = "*"
			}
			rows = append(rows, []string{marker, s.Name, s.ID, s.State})
		}
		printTable(stdout, []string{"CURRENT", "NAME", "ID", "STATE"}, rows)
	}
}

// cloudProviders is the priority-ordered provider list used for `auto` detection
// (azure → aws → gcp).
func cloudProviders() []cloud.Provider {
	return []cloud.Provider{azure.New(), aws.New(), gcp.New()}
}

// gather reads only the data sources required by the enabled segments. lookup
// is the environment view — the real env in render mode, the directive overlay
// in hook mode.
func gather(cfg config.Config, home string, lookup config.LookupEnv) render.Data {
	needKube := false
	needCloud := false
	for _, s := range cfg.Segments {
		switch s {
		case config.SegmentKube:
			// The kube display toggle (config `kube:` / OMNICTX_KUBE) gates the
			// segment on top of the segments list; namespace follows kube.
			needKube = cfg.Kube
		case config.SegmentCloud:
			needCloud = true
		}
	}

	var data render.Data
	if needCloud {
		if active, ok := cloud.Select(cloudProviders(), cfg.Cloud, cloud.LookupEnv(lookup), home); ok {
			if r := active.Read(cloud.LookupEnv(lookup), home); r.OK {
				data.Cloud = render.Cloud{
					Key:   active.Key(),
					Label: active.Label(cfg.Icons),
					Value: r.Text,
				}
			}
		}
	}
	// The namespace renders only as a suffix of the kube segment, so reading
	// the kubeconfig is only worthwhile when kube itself is enabled.
	if needKube {
		info := kube.Read(kube.LookupEnv(lookup), home)
		data.Kube = info.Context
		data.Namespace = info.Namespace
	}
	return data
}
