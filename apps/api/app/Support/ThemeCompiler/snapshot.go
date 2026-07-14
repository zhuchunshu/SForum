package themecompiler

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	htmltemplate "html/template"
	"reflect"
	"sort"
	"strings"
)

// 超时只能停止等待，不能抢占 html/template 内正在执行的用户方法。全局槽位
// 把极端模板遗留的后台执行限制在固定数量，避免每个请求都产生新 goroutine。
const maxConcurrentTemplateExecutions = 128

const maxViewModelValues = 100_000

var renderExecutionSlots = make(chan struct{}, maxConcurrentTemplateExecutions)

// Snapshot is immutable after compilation. The parse trees and helper maps are
// intentionally private because html/template mutation APIs are not safe after
// publication.
type Snapshot struct {
	key     SnapshotKey
	entries map[string]*htmltemplate.Template
	infos   []TemplateInfo
	limits  Limits
}

func (s *Snapshot) Key() SnapshotKey {
	if s == nil {
		return SnapshotKey{}
	}
	return s.key
}

func (s *Snapshot) CompiledKey() CompiledTemplateKey {
	if s == nil {
		return CompiledTemplateKey{}
	}
	return CompiledTemplateKey{
		PackageDigest: s.key.PackageDigest, CompilerVersion: s.key.CompilerVersion,
	}
}

func (s *Snapshot) CompiledCacheKey() string {
	key := s.CompiledKey()
	if key == (CompiledTemplateKey{}) {
		return ""
	}
	return key.CompilerVersion + ":" + key.PackageDigest
}

func (s *Snapshot) RuntimeKey() string {
	if s == nil {
		return ""
	}
	return s.CompiledCacheKey() + ":" + s.key.BindingRevision
}

func (s *Snapshot) Templates() []TemplateInfo {
	if s == nil {
		return nil
	}
	output := append([]TemplateInfo(nil), s.infos...)
	sort.Slice(output, func(i, j int) bool { return output[i].Name < output[j].Name })
	return output
}

func (s *Snapshot) HasTemplate(name string) bool {
	if s == nil {
		return false
	}
	_, ok := s.entries[name]
	return ok
}

func (s *Snapshot) Render(ctx context.Context, name string, data any) (string, error) {
	if s == nil || ctx == nil {
		return "", fmt.Errorf("%w: snapshot and context are required", ErrInvalidInput)
	}
	entry, ok := s.entries[name]
	if !ok {
		return "", fmt.Errorf("%w: %s", ErrTemplateNotFound, name)
	}
	if err := validatePassiveViewModel(data); err != nil {
		return "", err
	}
	renderCtx := ctx
	cancel := func() {}
	if s.limits.RenderTimeout > 0 {
		renderCtx, cancel = context.WithTimeout(ctx, s.limits.RenderTimeout)
	}
	defer cancel()

	writer := &boundedWriter{ctx: renderCtx, max: s.limits.MaxOutputBytes}
	select {
	case renderExecutionSlots <- struct{}{}:
	case <-renderCtx.Done():
		return "", classifyRenderContextError(renderCtx.Err())
	}
	result := make(chan error, 1)
	go func() {
		defer func() { <-renderExecutionSlots }()
		result <- entry.ExecuteTemplate(writer, name, data)
	}()

	var err error
	select {
	case err = <-result:
		if err == nil {
			err = renderCtx.Err()
		}
	case <-renderCtx.Done():
		err = renderCtx.Err()
	}
	if err == nil {
		return writer.String(), nil
	}
	if errors.Is(err, ErrOutputLimit) {
		return "", ErrOutputLimit
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return "", ErrRenderTimeout
	}
	if errors.Is(err, context.Canceled) {
		return "", context.Canceled
	}
	if errors.Is(err, ErrHelperValueMissing) {
		return "", fmt.Errorf("%w: %v", ErrHelperValueMissing, err)
	}
	if errors.Is(err, ErrSafeHTMLRequired) {
		return "", fmt.Errorf("%w: %v", ErrSafeHTMLRequired, err)
	}
	message := err.Error()
	if strings.Contains(message, "map has no entry for key") || strings.Contains(message, "can't evaluate field") ||
		strings.Contains(message, "nil pointer evaluating") {
		return "", fmt.Errorf("%w: %v", ErrMissingValue, err)
	}
	return "", fmt.Errorf("%w: %v", ErrExecution, err)
}

// validatePassiveViewModel prevents templates from invoking niladic methods or
// receiving html/template trusted-content aliases. Page ViewModels are data,
// not an executable Host API; already-sanitized rich content uses the sealed
// SForum SafeHTML value and must pass through the explicit safeHTML helper.
func validatePassiveViewModel(data any) error {
	remaining := maxViewModelValues
	var inspect func(reflect.Value, int) error
	inspect = func(value reflect.Value, depth int) error {
		if !value.IsValid() {
			return nil
		}
		if depth > DefaultMaxCallDepth*2 {
			return fmt.Errorf("%w: value nesting exceeds the host limit", ErrInvalidViewModel)
		}
		remaining--
		if remaining < 0 {
			return fmt.Errorf("%w: value count exceeds the host limit", ErrInvalidViewModel)
		}
		for value.Kind() == reflect.Interface {
			if value.IsNil() {
				return nil
			}
			value = value.Elem()
		}
		typeOf := value.Type()
		if typeOf.NumMethod() != 0 || isGoTrustedContentType(typeOf) {
			return fmt.Errorf("%w: type %s exposes executable or trusted content", ErrInvalidViewModel, typeOf)
		}
		switch value.Kind() {
		case reflect.Bool, reflect.String,
			reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
			reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64,
			reflect.Float32, reflect.Float64:
			return nil
		case reflect.Pointer:
			if value.IsNil() {
				return nil
			}
			return inspect(value.Elem(), depth+1)
		case reflect.Struct:
			for index := 0; index < value.NumField(); index++ {
				if typeOf.Field(index).PkgPath != "" {
					continue
				}
				if err := inspect(value.Field(index), depth+1); err != nil {
					return err
				}
			}
			return nil
		case reflect.Slice, reflect.Array:
			for index := 0; index < value.Len(); index++ {
				if err := inspect(value.Index(index), depth+1); err != nil {
					return err
				}
			}
			return nil
		case reflect.Map:
			if typeOf.Key().Kind() != reflect.String {
				return fmt.Errorf("%w: map keys must be strings", ErrInvalidViewModel)
			}
			iterator := value.MapRange()
			for iterator.Next() {
				if err := inspect(iterator.Value(), depth+1); err != nil {
					return err
				}
			}
			return nil
		default:
			return fmt.Errorf("%w: unsupported type %s", ErrInvalidViewModel, typeOf)
		}
	}
	return inspect(reflect.ValueOf(data), 0)
}

func isGoTrustedContentType(value reflect.Type) bool {
	if value.PkgPath() != "html/template" {
		return false
	}
	switch value.Name() {
	case "CSS", "HTML", "HTMLAttr", "JS", "JSStr", "Srcset", "URL":
		return true
	default:
		return false
	}
}

func classifyRenderContextError(err error) error {
	if errors.Is(err, context.DeadlineExceeded) {
		return ErrRenderTimeout
	}
	if errors.Is(err, context.Canceled) {
		return context.Canceled
	}
	return err
}

type boundedWriter struct {
	ctx     context.Context
	buffer  bytes.Buffer
	max     int64
	written int64
}

func (w *boundedWriter) Write(value []byte) (int, error) {
	if err := w.ctx.Err(); err != nil {
		return 0, err
	}
	if len(value) == 0 {
		return 0, nil
	}
	remaining := w.max - w.written
	if remaining <= 0 {
		return 0, ErrOutputLimit
	}
	if int64(len(value)) > remaining {
		count, _ := w.buffer.Write(value[:remaining])
		w.written += int64(count)
		return count, ErrOutputLimit
	}
	count, err := w.buffer.Write(value)
	w.written += int64(count)
	return count, err
}

func (w *boundedWriter) String() string {
	return w.buffer.String()
}
