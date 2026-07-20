package extensionsruntime

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"html/template"
	"os"
	"path/filepath"
	"strings"
	"sync"

	extensions "github.com/zhuchunshu/sforum/apps/api/app/Models/Extensions"
	extensionmanifest "github.com/zhuchunshu/sforum/apps/api/app/Support/ExtensionManifest"
)

// PackageLocalComponentSSRRenderer 在 Host 进程内编译并执行插件/主题包的
// 声明式 SSR 模板。模板在 Publish 时按 exact digest 读入并缓存，请求路径
// 不再访问包文件系统。
//
// 这是生产 PluginRenderer 的第一阶段：覆盖 package-local html/template 片段，
// 不依赖 Protocol V2 子进程。子进程渲染可在后续阶段替换同一接口。
type PackageLocalComponentSSRRenderer struct {
	mu    sync.RWMutex
	byKey map[string]*packageLocalCompiledTemplate // packageDigest + "\x00" + templateID
}

type packageLocalCompiledTemplate struct {
	extensionID   string
	packageDigest string
	templateID    string
	path          string
	digest        string
	compiled      *template.Template
}

// packageLocalTemplateData 是模板执行的固定数据契约。
// 仅暴露 Props/Result/ChildrenHTML；插件模板不得触达 Host 内部状态。
type packageLocalTemplateData struct {
	Props        map[string]any
	Result       map[string]any
	ChildrenHTML template.HTML
	Contribution string
	TargetID     string
}

// NewPackageLocalComponentSSRRenderer 创建空的包本地 SSR 渲染器。
func NewPackageLocalComponentSSRRenderer() *PackageLocalComponentSSRRenderer {
	return &PackageLocalComponentSSRRenderer{byKey: map[string]*packageLocalCompiledTemplate{}}
}

// Publish 编译 extension 中所有组件引用到的 SSR 模板。digest 不匹配或
// 路径越界时 fail closed。同一 packageDigest 的重复 Publish 是幂等替换。
func (r *PackageLocalComponentSSRRenderer) Publish(extension extensions.Extension) error {
	if r == nil {
		return fmt.Errorf("%w: package-local SSR renderer is nil", ErrComponentCompositionInvalid)
	}
	packageDigest := strings.TrimSpace(extension.PackageDigest)
	if packageDigest == "" || !packageLocalDigestPattern(packageDigest) {
		return fmt.Errorf("%w: package digest is required", ErrComponentCompositionInvalid)
	}
	root := strings.TrimSpace(extension.PackagePath)
	if root == "" {
		return fmt.Errorf("%w: package path is required", ErrComponentCompositionInvalid)
	}
	needed := map[string]extensions.ManifestTemplate{}
	for _, component := range extension.Manifest.Components {
		templateID := strings.TrimSpace(component.SSRTemplate)
		if templateID == "" {
			continue
		}
		declared, ok := componentTemplate(extension, templateID)
		if !ok {
			return fmt.Errorf("%w: template %s is not declared", ErrComponentCompositionInvalid, templateID)
		}
		needed[templateID] = declared
	}
	compiled := make(map[string]*packageLocalCompiledTemplate, len(needed))
	for templateID, declared := range needed {
		body, err := packageLocalReadExact(root, declared.Path, declared.Digest)
		if err != nil {
			return fmt.Errorf("%w: template %s: %v", ErrComponentCompositionInvalid, templateID, err)
		}
		// 仅允许 . / 字母数字与基础控制结构；html/template 自带上下文转义。
		tpl, err := template.New(templateID).Option("missingkey=zero").Parse(string(body))
		if err != nil {
			return fmt.Errorf("%w: template %s parse: %v", ErrComponentCompositionInvalid, templateID, err)
		}
		key := packageLocalTemplateKey(packageDigest, templateID)
		compiled[key] = &packageLocalCompiledTemplate{
			extensionID: extension.ID, packageDigest: packageDigest,
			templateID: templateID, path: declared.Path, digest: declared.Digest,
			compiled: tpl,
		}
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.byKey == nil {
		r.byKey = map[string]*packageLocalCompiledTemplate{}
	}
	// 先移除该 digest 的旧条目，再写入新表，避免半更新。
	for key, item := range r.byKey {
		if item.packageDigest == packageDigest {
			delete(r.byKey, key)
		}
	}
	for key, item := range compiled {
		r.byKey[key] = item
	}
	return nil
}

// RemovePackage 删除指定 packageDigest 的全部已编译模板。
func (r *PackageLocalComponentSSRRenderer) RemovePackage(packageDigest string) {
	if r == nil {
		return
	}
	packageDigest = strings.TrimSpace(packageDigest)
	r.mu.Lock()
	defer r.mu.Unlock()
	for key, item := range r.byKey {
		if item.packageDigest == packageDigest {
			delete(r.byKey, key)
		}
	}
}

// RemoveExtension 删除某扩展 id 下全部包版本的模板（禁用/卸载时使用）。
func (r *PackageLocalComponentSSRRenderer) RemoveExtension(extensionID string) {
	if r == nil {
		return
	}
	extensionID = strings.TrimSpace(extensionID)
	r.mu.Lock()
	defer r.mu.Unlock()
	for key, item := range r.byKey {
		if item.extensionID == extensionID {
			delete(r.byKey, key)
		}
	}
}

// Count 返回当前缓存的模板数（测试/检查器用）。
func (r *PackageLocalComponentSSRRenderer) Count() int {
	if r == nil {
		return 0
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	return len(r.byKey)
}

// RenderComponent 实现 ComponentSSRRenderer。按 exact package digest + SSR
// 模板 id 查找已编译模板；未发布时 fail closed。
func (r *PackageLocalComponentSSRRenderer) RenderComponent(
	ctx context.Context,
	call ComponentRenderCall,
) (ComponentRenderResponse, error) {
	if r == nil {
		return ComponentRenderResponse{}, ErrComponentCompositionCrash
	}
	if ctx != nil && ctx.Err() != nil {
		return ComponentRenderResponse{}, ctx.Err()
	}
	if isHostCoreComponentArtifact(call.Artifact) {
		return coreComponentRenderResponse(call), nil
	}
	templateID := strings.TrimSpace(call.Contribution.SSRTemplate)
	packageDigest := strings.TrimSpace(call.Artifact.PackageDigest)
	if templateID == "" || packageDigest == "" {
		return ComponentRenderResponse{}, ErrComponentCompositionCrash
	}
	key := packageLocalTemplateKey(packageDigest, templateID)
	r.mu.RLock()
	item := r.byKey[key]
	r.mu.RUnlock()
	if item == nil || item.compiled == nil {
		return ComponentRenderResponse{}, ErrComponentCompositionCrash
	}
	// filter_props / filter_result 只改 Document，不产生 HTML 片段。
	// 包本地阶段默认透传输入；需要变换时由后续 Protocol V2 渲染器接管。
	action := strings.TrimSpace(call.Contribution.Action)
	if action == extensionmanifest.ComponentActionFilterProps {
		return ComponentRenderResponse{
			Artifact: call.Artifact,
			Document: cloneComponentDocumentMust(call.Props),
		}, nil
	}
	if action == extensionmanifest.ComponentActionFilterResult {
		return ComponentRenderResponse{
			Artifact: call.Artifact,
			Document: cloneComponentDocumentMust(call.Result),
		}, nil
	}
	childrenHTML, err := packageLocalChildrenHTML(call.Children)
	if err != nil {
		return ComponentRenderResponse{}, err
	}
	data := packageLocalTemplateData{
		Props: call.Props, Result: call.Result, ChildrenHTML: template.HTML(childrenHTML),
		Contribution: call.Contribution.ID, TargetID: call.TargetID,
	}
	var buf bytes.Buffer
	if err := item.compiled.Execute(&buf, data); err != nil {
		return ComponentRenderResponse{}, fmt.Errorf("%w: execute %s: %v", ErrComponentCompositionCrash, templateID, err)
	}
	html := buf.String()
	if strings.TrimSpace(html) == "" {
		return ComponentRenderResponse{}, ErrComponentCompositionCrash
	}
	// 插件贡献默认不声明 PrimaryContent；主题 L1 / Core fallback 保留主内容。
	return ComponentRenderResponse{
		Artifact:  call.Artifact,
		Document:  cloneComponentDocumentMust(call.Result),
		Fragments: []ComponentRenderFragment{{ReviewedHTML: html}},
	}, nil
}

func packageLocalTemplateKey(packageDigest, templateID string) string {
	return packageDigest + "\x00" + templateID
}

func packageLocalDigestPattern(value string) bool {
	if len(value) != 64 {
		return false
	}
	for _, r := range value {
		if (r < '0' || r > '9') && (r < 'a' || r > 'f') {
			return false
		}
	}
	return true
}

// packageLocalReadExact 读取包内相对路径并校验 sha256 digest。
func packageLocalReadExact(root, rel, wantDigest string) ([]byte, error) {
	rel = strings.TrimSpace(rel)
	if rel == "" || strings.Contains(rel, "\\") || strings.HasPrefix(rel, "/") ||
		strings.Contains(rel, ":") || strings.Contains(rel, "..") {
		return nil, fmt.Errorf("path is not package-relative")
	}
	clean := filepath.Clean(filepath.FromSlash(rel))
	if clean == "." || clean == ".." || strings.HasPrefix(clean, ".."+string(filepath.Separator)) {
		return nil, fmt.Errorf("path escapes package root")
	}
	full := filepath.Join(root, clean)
	// 二次确认仍在 root 下。
	if !strings.HasPrefix(full, filepath.Clean(root)+string(filepath.Separator)) && full != filepath.Clean(root) {
		return nil, fmt.Errorf("path escapes package root")
	}
	body, err := os.ReadFile(full)
	if err != nil {
		return nil, err
	}
	sum := sha256.Sum256(body)
	got := hex.EncodeToString(sum[:])
	if !strings.EqualFold(got, strings.TrimSpace(wantDigest)) {
		return nil, fmt.Errorf("digest mismatch")
	}
	return body, nil
}

// packageLocalChildrenHTML 将已渲染子段拼接为模板可用的 HTML。
// 子段 safeHTML 已在 normalize 阶段消毒/转义。
func packageLocalChildrenHTML(children []ComponentRenderSegment) (string, error) {
	if len(children) == 0 {
		return "", nil
	}
	var builder strings.Builder
	for _, child := range children {
		if err := writePackageLocalSegmentHTML(&builder, child); err != nil {
			return "", err
		}
	}
	return builder.String(), nil
}

func writePackageLocalSegmentHTML(builder *strings.Builder, segment ComponentRenderSegment) error {
	// HTML 字段已在 normalize 阶段消毒/转义，可安全拼接进模板 ChildrenHTML。
	if strings.TrimSpace(segment.HTML) != "" {
		builder.WriteString(segment.HTML)
	}
	for _, child := range segment.Children {
		if err := writePackageLocalSegmentHTML(builder, child); err != nil {
			return err
		}
	}
	return nil
}
