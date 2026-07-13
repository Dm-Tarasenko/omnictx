package kube

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"gopkg.in/yaml.v3"
)

// Sentinel errors returned by WriteNamespace so the CLI can distinguish the
// kubeconfig-state failures. All three map to a non-zero exit in interactive
// mode; render mode never calls WriteNamespace.
var (
	// ErrNoActiveContext means no current-context is resolvable from the files.
	ErrNoActiveContext = errors.New("no active kube-context is set")
	// ErrContextNotFound means the active context is not defined in any file.
	ErrContextNotFound = errors.New("active kube-context is not defined in any kubeconfig")
	// ErrUnlocatable means the context's block could not be safely edited
	// (flow-style mapping, empty context map, or an unparsable target).
	ErrUnlocatable = errors.New("could not locate the context block to edit")
)

// namespaceRE matches a DNS-1123 label: lowercase alphanumeric or '-', starting
// and ending with an alphanumeric character. Length is checked separately.
var namespaceRE = regexp.MustCompile(`^[a-z0-9]([-a-z0-9]*[a-z0-9])?$`)

// ValidNamespace reports whether name is a syntactically valid Kubernetes
// namespace (a DNS-1123 label): 1–63 chars, lowercase alphanumeric or '-',
// starting and ending alphanumeric. omnictx is offline and cannot verify the
// namespace exists on the cluster, so this syntax gate is the strongest safe
// check before writing to a kubeconfig it does not own.
func ValidNamespace(name string) bool {
	return len(name) <= 63 && namespaceRE.MatchString(name)
}

// ContextEntry is one context as listed by `kube list`: the name plus the
// cluster/user/namespace columns of the kubectl get-contexts table. Fields
// absent from the kubeconfig are empty strings.
type ContextEntry struct {
	Name      string
	Cluster   string
	AuthInfo  string
	Namespace string
}

// Contexts returns every context found across the kubeconfig file list,
// deduplicated by name (first definition wins), in file-then-definition
// order. Broken or missing files are skipped, mirroring Read.
func Contexts(lookup LookupEnv, home string) []ContextEntry {
	var entries []ContextEntry
	seen := map[string]bool{}
	for _, f := range resolveFiles(lookup, home) {
		kf, ok := parseFile(f)
		if !ok {
			continue
		}
		for _, c := range kf.Contexts {
			if c.Name == "" || seen[c.Name] {
				continue
			}
			seen[c.Name] = true
			entries = append(entries, ContextEntry{
				Name:      c.Name,
				Cluster:   c.Context.Cluster,
				AuthInfo:  c.Context.User,
				Namespace: c.Context.Namespace,
			})
		}
	}
	return entries
}

// Check probes the kubeconfig file list for problems worth telling an
// interactive user about: files that exist but cannot be parsed. Missing
// files are a normal state and produce no warning. Render mode never calls
// this — the prompt stays silent by design.
func Check(lookup LookupEnv, home string) []string {
	var problems []string
	for _, f := range resolveFiles(lookup, home) {
		data, err := os.ReadFile(f)
		if err != nil {
			continue
		}
		var kf kubeFile
		if err := yaml.Unmarshal(data, &kf); err != nil {
			problems = append(problems, fmt.Sprintf("%s is unparsable: %v", f, err))
		}
	}
	return problems
}

// writeTarget picks the file that owns current-context: the first file in the
// list that sets a non-empty value, else the first file. This mirrors both the
// Read merge rule and kubectl's behavior, so a switch is always visible to the
// next Read.
func writeTarget(files []string) string {
	for _, f := range files {
		if kf, ok := parseFile(f); ok && kf.CurrentContext != "" {
			return f
		}
	}
	return files[0]
}

// WriteContext sets `current-context: <context>` in the kubeconfig chosen by
// writeTarget. The kubeconfig is a file omnictx does not own, so the edit is
// deliberately conservative:
//
//   - parse-before-write: a file that cannot be read or parsed is never touched;
//   - single-line surgery on the top-level current-context: line (appended when
//     absent) — every other byte, including comments, is preserved;
//   - atomic replace: temp file in the same directory with the original
//     permission bits, then rename.
//
// Validating that the context exists is the caller's job (see Contexts).
func WriteContext(lookup LookupEnv, home, context string) error {
	target := writeTarget(resolveFiles(lookup, home))

	data, err := os.ReadFile(target)
	if err != nil {
		return fmt.Errorf("read %s: %w", target, err)
	}
	var kf kubeFile
	if err := yaml.Unmarshal(data, &kf); err != nil {
		return fmt.Errorf("refusing to modify unparsable %s: %v", target, err)
	}

	newLine := "current-context: " + context
	lines := strings.Split(string(data), "\n")
	replaced := false
	for i, l := range lines {
		// Column-0 prefix only: an indented current-context: belongs to some
		// nested block, never to the document root.
		if strings.HasPrefix(l, "current-context:") {
			lines[i] = newLine
			replaced = true
			break
		}
	}
	out := strings.Join(lines, "\n")
	if !replaced {
		if out != "" && !strings.HasSuffix(out, "\n") {
			out += "\n"
		}
		out += newLine + "\n"
	}

	return atomicWrite(target, out)
}

// namespaceWriteTarget returns the first file whose contexts[] defines an entry
// named ctx, mirroring the read-side namespace resolution (first match wins).
// Unreadable or broken files are skipped. ok=false means no file defines ctx.
func namespaceWriteTarget(files []string, ctx string) (string, bool) {
	for _, f := range files {
		kf, ok := parseFile(f)
		if !ok {
			continue
		}
		for _, c := range kf.Contexts {
			if c.Name == ctx {
				return f, true
			}
		}
	}
	return "", false
}

// WriteNamespace sets the namespace of the active kube-context (the contexts[]
// entry matching current-context) to namespace, editing only that context's
// block in the kubeconfig that defines it. Like WriteContext, the edit is
// conservative — parse-before-write, byte-preserving surgery scoped to one
// block (replace the existing namespace value keeping any inline comment, or
// insert a namespace line as the first child of the context mapping), then an
// atomic replace preserving the original permission bits. Validating the name
// is the caller's job (see ValidNamespace).
func WriteNamespace(lookup LookupEnv, home, namespace string) error {
	files := resolveFiles(lookup, home)

	ctx := Read(lookup, home).Context
	if ctx == "" {
		return ErrNoActiveContext
	}
	target, ok := namespaceWriteTarget(files, ctx)
	if !ok {
		return ErrContextNotFound
	}

	data, err := os.ReadFile(target)
	if err != nil {
		return fmt.Errorf("read %s: %w", target, err)
	}
	var root yaml.Node
	if err := yaml.Unmarshal(data, &root); err != nil {
		// target came from parseFile, so this is unexpected; refuse regardless.
		return fmt.Errorf("refusing to modify unparsable %s: %w", target, ErrUnlocatable)
	}

	lines := strings.Split(string(data), "\n")
	lines, err = setNamespaceLines(lines, &root, ctx, namespace)
	if err != nil {
		return err
	}
	return atomicWrite(target, strings.Join(lines, "\n"))
}

// setNamespaceLines edits raw kubeconfig lines to set the namespace of the
// context named ctx to ns, using position info from the parsed node tree to
// locate the exact line/column. It replaces an existing namespace value
// (preserving a trailing inline comment) or inserts a new namespace line as the
// first child of the context mapping. It returns ErrUnlocatable for structures
// it cannot safely edit (flow-style mappings, empty context maps).
func setNamespaceLines(lines []string, root *yaml.Node, ctx, ns string) ([]string, error) {
	doc := documentRoot(root)
	contexts := mapValue(doc, "contexts")
	if contexts == nil || contexts.Kind != yaml.SequenceNode {
		return nil, ErrUnlocatable
	}

	var ctxMap *yaml.Node
	for _, item := range contexts.Content {
		if item.Kind != yaml.MappingNode {
			continue
		}
		if name := mapValue(item, "name"); name == nil || name.Value != ctx {
			continue
		}
		ctxMap = mapValue(item, "context")
		break
	}
	if ctxMap == nil || ctxMap.Kind != yaml.MappingNode || ctxMap.Style&yaml.FlowStyle != 0 {
		return nil, ErrUnlocatable
	}

	// Replace an existing namespace value in place.
	for i := 0; i+1 < len(ctxMap.Content); i += 2 {
		if ctxMap.Content[i].Value != "namespace" {
			continue
		}
		idx := ctxMap.Content[i].Line - 1
		if idx < 0 || idx >= len(lines) {
			return nil, ErrUnlocatable
		}
		newLine, ok := replaceNamespaceValue(lines[idx], ns)
		if !ok {
			return nil, ErrUnlocatable
		}
		lines[idx] = newLine
		return lines, nil
	}

	// No namespace key: insert one as the first child of the context mapping,
	// aligned with the existing siblings' indentation.
	if len(ctxMap.Content) == 0 {
		return nil, ErrUnlocatable
	}
	first := ctxMap.Content[0]
	at := first.Line - 1
	if at < 0 || at > len(lines) {
		return nil, ErrUnlocatable
	}
	newLine := strings.Repeat(" ", first.Column-1) + "namespace: " + ns
	out := make([]string, 0, len(lines)+1)
	out = append(out, lines[:at]...)
	out = append(out, newLine)
	out = append(out, lines[at:]...)
	return out, nil
}

// documentRoot unwraps a DocumentNode to its top-level content node.
func documentRoot(n *yaml.Node) *yaml.Node {
	if n != nil && n.Kind == yaml.DocumentNode {
		if len(n.Content) == 0 {
			return nil
		}
		return n.Content[0]
	}
	return n
}

// mapValue returns the value node for key in a mapping node, or nil.
func mapValue(m *yaml.Node, key string) *yaml.Node {
	if m == nil || m.Kind != yaml.MappingNode {
		return nil
	}
	for i := 0; i+1 < len(m.Content); i += 2 {
		if m.Content[i].Value == key {
			return m.Content[i+1]
		}
	}
	return nil
}

// replaceNamespaceValue rewrites the value of a `namespace:` line to ns while
// preserving indentation and any trailing inline comment. A namespace value is
// a DNS-1123 label (no '#', no spaces), so the first '#' after the key marks a
// comment. ok=false if the line has no key separator.
func replaceNamespaceValue(line, ns string) (string, bool) {
	colon := strings.Index(line, ":")
	if colon < 0 {
		return "", false
	}
	out := line[:colon+1] + " " + ns
	if h := strings.Index(line[colon+1:], "#"); h >= 0 {
		out += " " + strings.TrimSpace(line[colon+1+h:])
	}
	return out, true
}

// atomicWrite replaces path with content via a same-directory temp file and
// rename, preserving the original permission bits (kubeconfigs are usually
// 0600). A failure at any step leaves the original file untouched.
func atomicWrite(path, content string) error {
	mode := os.FileMode(0o600)
	if fi, err := os.Stat(path); err == nil {
		mode = fi.Mode().Perm()
	}

	tmp, err := os.CreateTemp(filepath.Dir(path), ".omnictx-*")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	defer func() { _ = os.Remove(tmpName) }() // no-op once renamed

	if _, err := tmp.WriteString(content); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Chmod(mode); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmpName, path)
}
