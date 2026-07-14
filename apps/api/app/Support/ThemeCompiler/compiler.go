package themecompiler

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	htmltemplate "html/template"
	"io"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

const internalRootTemplate = "__sforum_theme_root__"

var canonicalDigestPattern = regexp.MustCompile(`^[0-9a-f]{64}$`)

type Compiler struct {
	limits Limits
}

func NewCompiler(limits Limits) *Compiler {
	return &Compiler{limits: limits.normalized()}
}

func (c *Compiler) Version() string {
	return CompilerVersion
}

// CompileDir resolves the package root once. The returned snapshot retains no
// path or filesystem handle, so Render cannot perform request-time disk I/O.
func (c *Compiler) CompileDir(packageRoot, packageDigest string, bindings Bindings) (*Snapshot, error) {
	packageRoot = strings.TrimSpace(packageRoot)
	if packageRoot == "" {
		return nil, fmt.Errorf("%w: package root", ErrInvalidInput)
	}
	root, err := filepath.Abs(packageRoot)
	if err != nil {
		return nil, fmt.Errorf("%w: package root: %v", ErrInvalidInput, err)
	}
	realRoot, err := filepath.EvalSymlinks(root)
	if err != nil {
		return nil, fmt.Errorf("%w: resolve package root: %v", ErrInvalidInput, err)
	}
	info, err := os.Stat(realRoot)
	if err != nil || !info.IsDir() {
		return nil, fmt.Errorf("%w: package root is not a directory", ErrInvalidInput)
	}
	return c.CompileFS(os.DirFS(realRoot), packageDigest, bindings)
}

func (c *Compiler) CompileFS(source fs.FS, packageDigest string, bindings Bindings) (*Snapshot, error) {
	if c == nil || source == nil {
		return nil, fmt.Errorf("%w: compiler and source filesystem are required", ErrInvalidInput)
	}
	if !canonicalDigestPattern.MatchString(packageDigest) {
		return nil, ErrInvalidDigest
	}
	if !canonicalDigestPattern.MatchString(bindings.BindingRevision) {
		return nil, ErrInvalidBindingRevision
	}

	sources, err := loadSources(source, c.limits)
	if err != nil {
		return nil, err
	}
	pageCount := 0
	for _, item := range sources {
		if item.kind == KindPage {
			pageCount++
		}
	}
	if pageCount == 0 {
		return nil, ErrNoTemplates
	}

	immutableBindings := cloneBindings(bindings)
	funcs := restrictedFuncMap(immutableBindings)
	shared := htmltemplate.New(internalRootTemplate).Option("missingkey=error").Funcs(funcs)
	sharedNames := map[string]string{}
	for _, item := range sources {
		if item.kind == KindPage {
			continue
		}
		if err := mergeSharedSource(shared, sharedNames, item, funcs); err != nil {
			return nil, err
		}
	}

	entries := make(map[string]*htmltemplate.Template, pageCount)
	for _, item := range sources {
		if item.kind != KindPage {
			continue
		}
		if err := inspectStaticHTML(item.name, item.body); err != nil {
			return nil, err
		}
		compiled, err := shared.Clone()
		if err != nil {
			return nil, fmt.Errorf("%w: clone shared templates: %v", ErrInvalidTemplate, err)
		}
		compiled, err = compiled.New(item.name).Parse(string(item.body))
		if err != nil {
			return nil, classifyParseError(item.name, err)
		}
		if err := validateTemplateSet(compiled, item.name, c.limits.MaxCallDepth); err != nil {
			return nil, err
		}
		if err := validateContextEscaping(compiled, item.name); err != nil {
			return nil, err
		}
		entries[item.name] = compiled
	}

	infos := make([]TemplateInfo, 0, len(sources))
	for _, item := range sources {
		sum := sha256.Sum256(item.body)
		infos = append(infos, TemplateInfo{
			Name: item.name, Kind: item.kind, Digest: hex.EncodeToString(sum[:]), Bytes: int64(len(item.body)),
		})
	}
	return &Snapshot{
		key: SnapshotKey{
			PackageDigest: packageDigest, CompilerVersion: CompilerVersion,
			BindingRevision: immutableBindings.BindingRevision,
		},
		entries: entries, infos: infos, limits: c.limits,
	}, nil
}

type templateSource struct {
	name string
	kind TemplateKind
	body []byte
}

func loadSources(source fs.FS, limits Limits) ([]templateSource, error) {
	roots := []struct {
		name string
		kind TemplateKind
	}{
		{name: "layouts", kind: KindLayout},
		{name: "partials", kind: KindPartial},
		{name: "templates", kind: KindPage},
	}
	items := make([]templateSource, 0)
	var total int64
	for _, root := range roots {
		err := fs.WalkDir(source, root.name, func(name string, entry fs.DirEntry, walkErr error) error {
			if walkErr != nil {
				if name == root.name && errors.Is(walkErr, fs.ErrNotExist) {
					return fs.SkipDir
				}
				return walkErr
			}
			if entry.Type()&os.ModeSymlink != 0 {
				return fmt.Errorf("%w: symlink %s", ErrInvalidTemplate, name)
			}
			if entry.IsDir() || path.Ext(name) != ".html" {
				return nil
			}
			if !entry.Type().IsRegular() {
				return fmt.Errorf("%w: non-regular source %s", ErrInvalidTemplate, name)
			}
			if err := validateTemplateName(name); err != nil {
				return err
			}
			if len(items) >= limits.MaxFiles {
				return fmt.Errorf("%w: more than %d source files", ErrInvalidTemplate, limits.MaxFiles)
			}
			body, err := readBoundedFile(source, name, limits.MaxSourceBytes)
			if err != nil {
				return err
			}
			total += int64(len(body))
			if total > limits.MaxTotalBytes {
				return fmt.Errorf("%w: total source exceeds %d bytes", ErrInvalidTemplate, limits.MaxTotalBytes)
			}
			items = append(items, templateSource{name: name, kind: root.kind, body: body})
			return nil
		})
		if err != nil && !errors.Is(err, fs.ErrNotExist) {
			return nil, fmt.Errorf("%w: load %s: %w", ErrInvalidTemplate, root.name, err)
		}
	}
	sort.Slice(items, func(i, j int) bool { return items[i].name < items[j].name })
	return items, nil
}

func readBoundedFile(source fs.FS, name string, limit int64) ([]byte, error) {
	file, err := source.Open(name)
	if err != nil {
		return nil, fmt.Errorf("%w: open %s: %v", ErrInvalidTemplate, name, err)
	}
	defer file.Close()
	body, err := io.ReadAll(io.LimitReader(file, limit+1))
	if err != nil {
		return nil, fmt.Errorf("%w: read %s: %v", ErrInvalidTemplate, name, err)
	}
	if int64(len(body)) > limit {
		return nil, fmt.Errorf("%w: %s exceeds %d bytes", ErrInvalidTemplate, name, limit)
	}
	return body, nil
}

func mergeSharedSource(root *htmltemplate.Template, names map[string]string, item templateSource, funcs htmltemplate.FuncMap) error {
	if err := inspectStaticHTML(item.name, item.body); err != nil {
		return err
	}
	parsed, err := htmltemplate.New(item.name).Option("missingkey=error").Funcs(funcs).Parse(string(item.body))
	if err != nil {
		return classifyParseError(item.name, err)
	}
	for _, candidate := range parsed.Templates() {
		if candidate.Tree == nil {
			continue
		}
		name := candidate.Name()
		if err := validateTemplateName(name); err != nil {
			return err
		}
		if previous, exists := names[name]; exists {
			return fmt.Errorf("%w: template %q defined by both %s and %s", ErrInvalidPartial, name, previous, item.name)
		}
		names[name] = item.name
		if _, err := root.AddParseTree(name, candidate.Tree.Copy()); err != nil {
			return fmt.Errorf("%w: add %s: %v", ErrInvalidTemplate, name, err)
		}
	}
	return nil
}

func classifyParseError(name string, err error) error {
	if strings.Contains(err.Error(), "function ") && strings.Contains(err.Error(), " not defined") {
		return fmt.Errorf("%w: %s: %v", ErrForbiddenHelper, name, err)
	}
	return fmt.Errorf("%w: parse %s: %v", ErrInvalidTemplate, name, err)
}

func validateTemplateName(name string) error {
	if name == "" || len(name) > 256 || strings.HasPrefix(name, "/") || strings.Contains(name, "\\") ||
		strings.Contains(name, "..") || !fs.ValidPath(name) {
		return fmt.Errorf("%w: invalid template name %q", ErrInvalidPartial, name)
	}
	return nil
}
