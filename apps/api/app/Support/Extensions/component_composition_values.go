package extensionsruntime

import (
	"bytes"
	"context"
	"encoding"
	"encoding/json"
	"errors"
	"fmt"
	htmltemplate "html/template"
	"math"
	"reflect"
	"regexp"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/microcosm-cc/bluemonday"
)

var componentReviewedHTMLPolicy = newComponentReviewedHTMLPolicy()

const (
	maximumComponentDocumentDepth = 64
	maximumComponentDocumentNodes = 100_000
)

type componentDocumentBudget struct {
	remaining int
	nodes     int
	maxNodes  int
}

type componentBoundedStringWriter struct {
	builder   strings.Builder
	remaining int
}

func (w *componentBoundedStringWriter) Write(value []byte) (int, error) {
	if w == nil || len(value) > w.remaining {
		return 0, ErrComponentCompositionOutput
	}
	written, err := w.builder.Write(value)
	w.remaining -= written
	return written, err
}

func newComponentReviewedHTMLPolicy() *bluemonday.Policy {
	policy := bluemonday.NewPolicy()
	policy.AllowElements(
		"a", "abbr", "article", "aside", "b", "blockquote", "br", "cite", "code",
		"dd", "div", "dl", "dt", "em", "figcaption", "figure", "footer", "h1",
		"h2", "h3", "h4", "h5", "h6", "header", "hr", "i", "img", "li", "main",
		"mark", "nav", "ol", "p", "pre", "section", "small", "span", "strong", "sub",
		"sup", "table", "tbody", "td", "tfoot", "th", "thead", "tr", "u", "ul",
	)
	policy.AllowAttrs("class", "id", "title", "role", "aria-label", "aria-hidden", "aria-describedby").Globally()
	policy.AllowAttrs("colspan", "rowspan", "scope").OnElements("td", "th")
	policy.AllowAttrs("start", "type").OnElements("ol")
	policy.AllowAttrs("alt", "width", "height", "loading").OnElements("img")
	policy.AllowAttrs("src").Matching(regexp.MustCompile(`^(https:)?//|^/[^/\\]|^[a-zA-Z0-9_./-]+$`)).OnElements("img")
	policy.AllowAttrs("href").Matching(regexp.MustCompile(`^(https?:)?//|^/|^#|^mailto:`)).OnElements("a")
	policy.AllowAttrs("rel", "target").OnElements("a")
	// bluemonday v1.0.27 does not include <main> in its built-in set of
	// elements valid without attributes, despite AllowElements accepting it.
	policy.AllowNoAttrs().OnElements("main")
	policy.RequireNoFollowOnLinks(true)
	policy.RequireNoReferrerOnLinks(true)
	policy.AllowStandardURLs()
	policy.RequireParseableURLs(true)
	policy.AllowURLSchemes("http", "https", "mailto")
	return policy
}

// preflightComponentDocumentsBounded conservatively budgets the canonical JSON
// tree before encoding/json can allocate a second complete document. Custom
// marshalers are rejected because their output and side effects are not bounded
// by the value graph inspected by the Host.
func preflightComponentDocumentsBounded(maximumBytes int, values ...map[string]any) error {
	if maximumBytes < 1 {
		return ErrComponentCompositionOutput
	}
	maxNodes := maximumBytes
	if maxNodes > maximumComponentDocumentNodes {
		maxNodes = maximumComponentDocumentNodes
	}
	budget := &componentDocumentBudget{remaining: maximumBytes, maxNodes: maxNodes}
	for _, value := range values {
		if value == nil {
			value = map[string]any{}
		}
		if err := budget.walk(reflect.ValueOf(value), 0); err != nil {
			return err
		}
	}
	return nil
}

func (b *componentDocumentBudget) walk(value reflect.Value, depth int) error {
	if depth > maximumComponentDocumentDepth {
		return fmt.Errorf("%w: document nesting or cycle exceeds Host bounds", ErrComponentCompositionOutput)
	}
	b.nodes++
	if b.nodes > b.maxNodes {
		return fmt.Errorf("%w: document node count exceeds Host bounds", ErrComponentCompositionOutput)
	}
	if !value.IsValid() {
		return b.consume(4)
	}
	if value.Kind() == reflect.Interface {
		if value.IsNil() {
			return b.consume(4)
		}
		return b.walk(value.Elem(), depth+1)
	}
	if customComponentDocumentMarshaler(value) {
		return fmt.Errorf("%w: custom JSON or text marshalers are not allowed", ErrComponentCompositionInvalid)
	}

	switch value.Kind() {
	case reflect.Pointer:
		if value.IsNil() {
			return b.consume(4)
		}
		return b.walk(value.Elem(), depth+1)
	case reflect.Map:
		if value.IsNil() {
			return b.consume(4)
		}
		if value.Type().Key().Kind() != reflect.String {
			return fmt.Errorf("%w: component documents require string object keys", ErrComponentCompositionInvalid)
		}
		if err := b.consume(2); err != nil {
			return err
		}
		iterator := value.MapRange()
		index := 0
		for iterator.Next() {
			if index > 0 {
				if err := b.consume(1); err != nil {
					return err
				}
			}
			key := iterator.Key()
			if customComponentDocumentMarshaler(key) {
				return fmt.Errorf("%w: custom object-key marshalers are not allowed", ErrComponentCompositionInvalid)
			}
			if err := b.consumeJSONString(key.String()); err != nil {
				return err
			}
			if err := b.consume(1); err != nil {
				return err
			}
			if err := b.walk(iterator.Value(), depth+1); err != nil {
				return err
			}
			index++
		}
		return nil
	case reflect.Slice:
		if value.IsNil() {
			return b.consume(4)
		}
		if value.Type().Elem().Kind() == reflect.Uint8 {
			return b.consumeByteSlice(value.Len())
		}
		fallthrough
	case reflect.Array:
		if err := b.consume(2); err != nil {
			return err
		}
		for index := 0; index < value.Len(); index++ {
			if index > 0 {
				if err := b.consume(1); err != nil {
					return err
				}
			}
			if err := b.walk(value.Index(index), depth+1); err != nil {
				return err
			}
		}
		return nil
	case reflect.String:
		return b.consumeJSONString(value.String())
	case reflect.Bool:
		return b.consume(5)
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		return b.consume(24)
	case reflect.Float32, reflect.Float64:
		if number := value.Float(); math.IsNaN(number) || math.IsInf(number, 0) {
			return fmt.Errorf("%w: non-finite component document number", ErrComponentCompositionInvalid)
		}
		return b.consume(32)
	default:
		return fmt.Errorf("%w: unsupported component document value %s", ErrComponentCompositionInvalid, value.Kind())
	}
}

func customComponentDocumentMarshaler(value reflect.Value) bool {
	implements := func(candidate reflect.Value) bool {
		if !candidate.IsValid() || !candidate.CanInterface() {
			return false
		}
		item := candidate.Interface()
		_, jsonCustom := item.(json.Marshaler)
		_, textCustom := item.(encoding.TextMarshaler)
		return jsonCustom || textCustom
	}
	if implements(value) {
		return true
	}
	// encoding/json uses pointer method sets for addressable slice/array values.
	// Detect those methods before Marshal can run plugin-controlled code.
	return value.CanAddr() && implements(value.Addr())
}

func (b *componentDocumentBudget) consumeByteSlice(length int) error {
	maximumInt := int(^uint(0) >> 1)
	if length < 0 || length > maximumInt-2 || b.remaining < 2 {
		return ErrComponentCompositionOutput
	}
	groups := (length + 2) / 3
	if groups > (b.remaining-2)/4 {
		return ErrComponentCompositionOutput
	}
	return b.consume(2 + 4*groups)
}

func (b *componentDocumentBudget) consumeJSONString(value string) error {
	if err := b.consume(2); err != nil {
		return err
	}
	for len(value) > 0 {
		current := value[0]
		switch {
		case current < 0x20 || current == '<' || current == '>' || current == '&':
			if err := b.consume(6); err != nil {
				return err
			}
			value = value[1:]
		case current == '\\' || current == '"':
			if err := b.consume(2); err != nil {
				return err
			}
			value = value[1:]
		case current < utf8.RuneSelf:
			if err := b.consume(1); err != nil {
				return err
			}
			value = value[1:]
		default:
			runeValue, size := utf8.DecodeRuneInString(value)
			if runeValue == utf8.RuneError && size == 1 || runeValue == '\u2028' || runeValue == '\u2029' {
				if err := b.consume(6); err != nil {
					return err
				}
			} else if err := b.consume(size); err != nil {
				return err
			}
			value = value[size:]
		}
	}
	return nil
}

func (b *componentDocumentBudget) consume(size int) error {
	if size < 0 || size > b.remaining {
		return ErrComponentCompositionOutput
	}
	b.remaining -= size
	return nil
}

func cloneComponentDocument(value map[string]any, maximumBytes int) (map[string]any, error) {
	if value == nil {
		value = map[string]any{}
	}
	if err := preflightComponentDocumentsBounded(maximumBytes, value); err != nil {
		return nil, err
	}
	raw, err := json.Marshal(value)
	if err != nil {
		return nil, fmt.Errorf("%w: encode component document: %v", ErrComponentCompositionInvalid, err)
	}
	if len(raw) > maximumBytes {
		return nil, ErrComponentCompositionOutput
	}
	result := make(map[string]any)
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.UseNumber()
	if err := decoder.Decode(&result); err != nil {
		return nil, err
	}
	return result, nil
}

func cloneComponentDocumentMust(value map[string]any) map[string]any {
	result, err := cloneComponentDocument(value, DefaultComponentCompositionMaxBytes)
	if err != nil {
		return map[string]any{}
	}
	return result
}

func cloneComponentRenderResponse(value ComponentRenderResponse, maximumBytes int) (ComponentRenderResponse, error) {
	if maximumBytes < 1 {
		return ComponentRenderResponse{}, ErrComponentCompositionOutput
	}
	inputBudget := maximumBytes
	for _, fragment := range value.Fragments {
		content := fragment.Text
		if fragment.ReviewedHTML != "" {
			content = fragment.ReviewedHTML
		}
		if len(content) > inputBudget {
			return ComponentRenderResponse{}, ErrComponentCompositionOutput
		}
		inputBudget -= len(content)
	}

	outputBudget := maximumBytes
	value.Fragments = append([]ComponentRenderFragment(nil), value.Fragments...)
	for index := range value.Fragments {
		fragment, err := normalizeComponentRenderFragment(value.Fragments[index], outputBudget)
		if err != nil {
			return ComponentRenderResponse{}, err
		}
		outputBudget -= len(fragment.safeHTML)
		value.Fragments[index] = fragment
	}
	var document map[string]any
	if value.Document != nil {
		var err error
		document, err = cloneComponentDocument(value.Document, outputBudget)
		if err != nil {
			return ComponentRenderResponse{}, err
		}
	}
	value.Document = document
	return value, nil
}

func cloneComponentRenderSegments(values []ComponentRenderSegment) []ComponentRenderSegment {
	result := make([]ComponentRenderSegment, len(values))
	for index, value := range values {
		value.Fallback = append([]ComponentFallbackEvidence(nil), value.Fallback...)
		value.Children = cloneComponentRenderSegments(value.Children)
		if value.Artifact != nil {
			artifact := *value.Artifact
			value.Artifact = &artifact
		}
		result[index] = value
	}
	return result
}

func componentSegmentsFromResponse(
	contribution ComponentContribution,
	response ComponentRenderResponse,
	children []ComponentRenderSegment,
	depth int,
	maxSegments int,
	maxBytes int,
) ([]ComponentRenderSegment, error) {
	if err := preflightComponentSegmentExpansion(response.Fragments, children, maxSegments, maxBytes); err != nil {
		return nil, err
	}
	result := make([]ComponentRenderSegment, len(response.Fragments))
	for index, fragment := range response.Fragments {
		artifact := contribution.Artifact
		result[index] = ComponentRenderSegment{
			OwnerID: contribution.Artifact.ExtensionID, ComponentID: contribution.ID,
			ContractVersion: contribution.ContractVersion, Action: contribution.Action,
			Depth: depth, HTML: fragment.safeHTML, Encoding: fragment.encoding,
			PrimaryContent: fragment.PrimaryContent,
			L2Component:    contribution.L2Component, Artifact: &artifact,
			Children: cloneComponentRenderSegments(children),
		}
	}
	return result, nil
}

func coreComponentSegments(
	target ComponentTarget,
	fragments []ComponentRenderFragment,
	depth int,
) []ComponentRenderSegment {
	result := make([]ComponentRenderSegment, len(fragments))
	for index, fragment := range fragments {
		result[index] = ComponentRenderSegment{
			OwnerID: "core", ComponentID: target.ID, ContractVersion: target.ContractVersion,
			Action: "fallback", Depth: depth, HTML: fragment.safeHTML,
			Encoding:       fragment.encoding,
			PrimaryContent: fragment.PrimaryContent,
		}
	}
	return result
}

func validateComponentBinding(plan ComponentResolvePlan, binding ComponentTargetBinding) error {
	if binding.Contract.ValidateProps == nil || binding.Contract.ValidateResult == nil {
		return fmt.Errorf("%w: target validators are required", ErrComponentCompositionInvalid)
	}
	if err := validateComponentMutableFieldList(binding.Contract.MutablePropsFields); err != nil {
		return err
	}
	if err := validateComponentMutableFieldList(binding.Contract.MutableResultFields); err != nil {
		return err
	}
	if plan.Target.ID == "" || plan.Target.ContractVersion == "" {
		return ErrComponentCompositionInvalid
	}
	return nil
}

func validateComponentMutableFieldList(fields []string) error {
	seen := make(map[string]struct{}, len(fields))
	for _, field := range fields {
		if field == "" || field != strings.TrimSpace(field) {
			return fmt.Errorf("%w: invalid mutable field", ErrComponentCompositionInvalid)
		}
		if _, duplicate := seen[field]; duplicate {
			return fmt.Errorf("%w: duplicate mutable field", ErrComponentCompositionInvalid)
		}
		seen[field] = struct{}{}
	}
	return nil
}

func enforceComponentMutableFields(current, candidate map[string]any, mutable []string) error {
	if candidate == nil {
		return ErrComponentCompositionInvalid
	}
	allowed := make(map[string]struct{}, len(mutable))
	for _, field := range mutable {
		allowed[field] = struct{}{}
	}
	keys := make(map[string]struct{}, len(current)+len(candidate))
	for key := range current {
		keys[key] = struct{}{}
	}
	for key := range candidate {
		keys[key] = struct{}{}
	}
	for key := range keys {
		if reflect.DeepEqual(current[key], candidate[key]) {
			continue
		}
		if _, ok := allowed[key]; !ok {
			return fmt.Errorf("%w: %s", ErrComponentCompositionMutation, key)
		}
	}
	return nil
}

func validateComponentFragmentBounds(fragments []ComponentRenderFragment, maxSegments, maxBytes int) error {
	if len(fragments) > maxSegments || maxBytes < 1 {
		return ErrComponentCompositionOutput
	}
	bytes := 0
	for _, fragment := range fragments {
		if (fragment.Text == "") == (fragment.ReviewedHTML == "") {
			return ErrComponentCompositionOutput
		}
		content := fragment.Text
		if fragment.ReviewedHTML != "" {
			content = fragment.ReviewedHTML
		}
		if len(content) > maxBytes-bytes {
			return ErrComponentCompositionOutput
		}
		bytes += len(content)
		if fragment.PrimaryContent && strings.TrimSpace(content) == "" {
			return ErrComponentCompositionOutput
		}
	}
	return nil
}

func normalizeComponentRenderFragment(
	fragment ComponentRenderFragment,
	maximumBytes int,
) (ComponentRenderFragment, error) {
	if (fragment.Text == "") == (fragment.ReviewedHTML == "") {
		return ComponentRenderFragment{}, ErrComponentCompositionOutput
	}
	if fragment.Text != "" {
		if _, err := componentEscapedTextLength(fragment.Text, maximumBytes); err != nil {
			return ComponentRenderFragment{}, err
		}
		fragment.safeHTML = htmltemplate.HTMLEscapeString(fragment.Text)
		fragment.encoding = ComponentRenderEncodingEscapedText
	} else {
		writer := &componentBoundedStringWriter{remaining: maximumBytes}
		if err := componentReviewedHTMLPolicy.SanitizeReaderToWriter(
			strings.NewReader(fragment.ReviewedHTML), writer,
		); err != nil {
			return ComponentRenderFragment{}, ErrComponentCompositionOutput
		}
		fragment.safeHTML = writer.builder.String()
		fragment.encoding = ComponentRenderEncodingSanitizedHTML
	}
	if fragment.PrimaryContent && strings.TrimSpace(fragment.safeHTML) == "" {
		return ComponentRenderFragment{}, ErrComponentCompositionOutput
	}
	return fragment, nil
}

func componentEscapedTextLength(value string, maximumBytes int) (int, error) {
	size := 0
	for index := 0; index < len(value); index++ {
		encoded := 1
		switch value[index] {
		case 0:
			encoded = len("\uFFFD")
		case '"', '\'', '&':
			encoded = 5
		case '<', '>':
			encoded = 4
		}
		if encoded > maximumBytes-size {
			return 0, ErrComponentCompositionOutput
		}
		size += encoded
	}
	return size, nil
}

// preflightComponentSegmentExpansion computes the complete wrap multiplication
// before cloning children. A renderer returning N wrapper fragments around M
// child nodes therefore cannot allocate N copies unless the final tree already
// fits both Host ceilings.
func preflightComponentSegmentExpansion(
	fragments []ComponentRenderFragment,
	children []ComponentRenderSegment,
	maxSegments int,
	maxBytes int,
) error {
	childCount, childBytes := componentSegmentSize(children)
	fragmentCount := len(fragments)
	if fragmentCount > maxSegments || childCount > maxSegments {
		return ErrComponentCompositionOutput
	}
	if fragmentCount > 0 && childCount > (maxSegments-fragmentCount)/fragmentCount {
		return ErrComponentCompositionOutput
	}
	fragmentBytes := 0
	for _, fragment := range fragments {
		if len(fragment.safeHTML) > maxBytes-fragmentBytes {
			return ErrComponentCompositionOutput
		}
		fragmentBytes += len(fragment.safeHTML)
	}
	if fragmentCount > 0 && childBytes > (maxBytes-fragmentBytes)/fragmentCount {
		return ErrComponentCompositionOutput
	}
	return nil
}

func validateComponentOutputBounds(
	segments []ComponentRenderSegment,
	props map[string]any,
	result map[string]any,
	maxSegments, maxBytes int,
) error {
	segmentCount, segmentBytes := componentSegmentSize(segments)
	if segmentCount > maxSegments || segmentBytes > maxBytes {
		return ErrComponentCompositionOutput
	}
	return preflightComponentDocumentsBounded(maxBytes-segmentBytes, props, result)
}

func componentSegmentSize(segments []ComponentRenderSegment) (int, int) {
	count, bytes := 0, 0
	for _, segment := range segments {
		childCount, childBytes := componentSegmentSize(segment.Children)
		count += 1 + childCount
		bytes += len(segment.HTML) + childBytes
	}
	return count, bytes
}

func segmentsHavePrimaryContent(segments []ComponentRenderSegment) bool {
	for _, segment := range segments {
		if segment.PrimaryContent && strings.TrimSpace(segment.HTML) != "" ||
			segmentsHavePrimaryContent(segment.Children) {
			return true
		}
	}
	return false
}

func retainPrimaryComponentSegments(segments []ComponentRenderSegment) []ComponentRenderSegment {
	result := make([]ComponentRenderSegment, 0, len(segments))
	for _, segment := range segments {
		if segment.PrimaryContent && strings.TrimSpace(segment.HTML) != "" ||
			segmentsHavePrimaryContent(segment.Children) {
			segment.Children = retainPrimaryComponentSegments(segment.Children)
			result = append(result, segment)
		}
	}
	return result
}

func addComponentFallbackEvidence(segments []ComponentRenderSegment, evidence []ComponentFallbackEvidence) {
	if len(evidence) == 0 {
		return
	}
	for index := range segments {
		segments[index].Fallback = append(segments[index].Fallback, evidence...)
	}
}

func appendFallback(
	values []ComponentFallbackEvidence,
	value *ComponentFallbackEvidence,
) []ComponentFallbackEvidence {
	if value == nil {
		return values
	}
	return append(values, *value)
}

func assignComponentSegmentOrder(segments []ComponentRenderSegment, order *int) {
	for index := range segments {
		segments[index].Order = *order
		*order = *order + 1
		assignComponentSegmentOrder(segments[index].Children, order)
	}
}

func errorsIsComponentStale(err error) bool {
	return errors.Is(err, ErrComponentCompositionStale) ||
		errors.Is(err, ErrComponentRegistryRevisionConflict)
}

func componentFailureReason(err error) string {
	switch {
	case errors.Is(err, ErrComponentCompositionStale):
		return "stale_snapshot"
	case errors.Is(err, ErrComponentCompositionUnauthorized):
		return "unauthorized_artifact"
	case errors.Is(err, ErrComponentCompositionTimeout):
		return "timeout"
	case errors.Is(err, ErrComponentCompositionCrash):
		return "crash"
	case errors.Is(err, ErrComponentCompositionMutation):
		return "forbidden_mutation"
	case errors.Is(err, ErrComponentCompositionSEO):
		return "seo_content_removed"
	case errors.Is(err, context.Canceled), errors.Is(err, context.DeadlineExceeded):
		return "cancelled"
	case errors.Is(err, ErrComponentCompositionOutput):
		return "output_limit"
	case errors.Is(err, ErrComponentCompositionBusy):
		return "renderer_busy"
	default:
		return "renderer_failure"
	}
}

func (e *ComponentCompositionExecutor) newTrace(request ComponentCompositionRequest) *ComponentCompositionTrace {
	sequence := uint64(0)
	if e != nil {
		sequence = e.traceSequence.Add(1)
	}
	return &ComponentCompositionTrace{
		ID: fmt.Sprintf("component-compose-%d", sequence), TargetID: strings.TrimSpace(request.TargetID),
		TargetContractVersion: strings.TrimSpace(request.TargetContractVersion), StartedAt: time.Now().UTC(),
	}
}

func (r *componentCompositionRun) addTraceStep(
	targetID string,
	contribution ComponentContribution,
	policy ComponentCallPolicy,
	status, fallbackReason string,
	duration time.Duration,
) {
	r.trace.Steps = append(r.trace.Steps, ComponentCompositionTraceStep{
		Sequence: len(r.trace.Steps), TargetID: targetID, ContributionID: contribution.ID,
		Action: contribution.Action, Artifact: contribution.Artifact, Status: status,
		FailurePolicy: policy.FailurePolicy, TimeoutMS: policy.Timeout.Milliseconds(),
		FallbackReason: fallbackReason, DurationMicros: duration.Microseconds(),
	})
}

func (r *componentCompositionRun) addCoreTraceStep(
	targetID, status, reason string,
	duration time.Duration,
) {
	r.trace.Steps = append(r.trace.Steps, ComponentCompositionTraceStep{
		Sequence: len(r.trace.Steps), TargetID: targetID, Action: "fallback", Status: status,
		TimeoutMS: r.executor.defaultTimeout.Milliseconds(), FallbackReason: reason,
		DurationMicros: duration.Microseconds(),
	})
}

func (e *ComponentCompositionExecutor) recordTrace(trace ComponentCompositionTrace) {
	if e == nil {
		return
	}
	trace.Steps = append([]ComponentCompositionTraceStep(nil), trace.Steps...)
	e.traceMu.Lock()
	defer e.traceMu.Unlock()
	if len(e.traces) == e.traceLimit {
		copy(e.traces, e.traces[1:])
		e.traces[len(e.traces)-1] = trace
		return
	}
	e.traces = append(e.traces, trace)
}

func (e *ComponentCompositionExecutor) InspectorTraces() []ComponentCompositionTrace {
	if e == nil {
		return nil
	}
	e.traceMu.Lock()
	defer e.traceMu.Unlock()
	result := make([]ComponentCompositionTrace, len(e.traces))
	for index, trace := range e.traces {
		trace.Steps = append([]ComponentCompositionTraceStep(nil), trace.Steps...)
		result[index] = trace
	}
	return result
}
