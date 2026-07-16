package contentregistry

import "fmt"

func preflightRenderSegmentsJSON(input RenderSegments, limits ExecutionLimits) error {
	remaining := limits.MaxOutputBytes
	if !consumeRenderSegmentsJSONBudget(&remaining, input, limits) {
		return ErrExecutionLimit
	}
	return nil
}

func consumeRenderSegmentsJSONBudget(remaining *int, input RenderSegments, limits ExecutionLimits) bool {
	if len(input.Segments) > limits.MaxSegments || !consumeExecutionJSONBudget(remaining, 160) ||
		!consumeExecutionJSONString(remaining, input.SchemaVersion) ||
		!consumeExecutionJSONString(remaining, input.ContentID) ||
		!consumeExecutionJSONString(remaining, input.ContractVersion) ||
		!consumeExecutionJSONString(remaining, input.TextEncoding) ||
		!consumeExecutionJSONString(remaining, input.PlainText) {
		return false
	}
	for _, segment := range input.Segments {
		if !consumeRenderSegmentJSONBudget(remaining, segment) {
			return false
		}
	}
	return true
}

func consumeRenderSegmentJSONBudget(remaining *int, segment RenderSegment) bool {
	return consumeExecutionJSONBudget(remaining, 48) &&
		consumeExecutionJSONString(remaining, segment.Kind) &&
		consumeExecutionJSONString(remaining, segment.HTML) &&
		consumeExecutionJSONString(remaining, segment.Text)
}

func preflightMergedRenderSegments(target Contribution, limits ExecutionLimits, groups ...RenderSegments) error {
	remaining := limits.MaxOutputBytes
	if !consumeExecutionJSONBudget(&remaining, 160) ||
		!consumeExecutionJSONString(&remaining, RenderSegmentsSchemaVersion) ||
		!consumeExecutionJSONString(&remaining, target.ID) ||
		!consumeExecutionJSONString(&remaining, target.ContractVersion) ||
		!consumeExecutionJSONString(&remaining, "") {
		return ErrExecutionLimit
	}
	count := 0
	if !consumeExecutionJSONBudget(&remaining, 2) {
		return ErrExecutionLimit
	}
	for _, group := range groups {
		for _, segment := range group.Segments {
			count++
			if count > limits.MaxSegments || !consumeRenderSegmentJSONBudget(&remaining, segment) {
				return ErrExecutionLimit
			}
			// PlainText is rebuilt from these fields. Consuming the raw source
			// again is conservative and avoids allocating a combined slice first.
			if !consumeExecutionJSONString(&remaining, segment.HTML) ||
				!consumeExecutionJSONString(&remaining, segment.Text) {
				return ErrExecutionLimit
			}
		}
	}
	return nil
}

func preflightExecutionResultJSON(result ExecutionResult, limits ExecutionLimits) error {
	remaining := limits.MaxOutputBytes
	if !consumeExecutionJSONBudget(&remaining, 384) ||
		!consumeExecutionJSONString(&remaining, result.SchemaVersion) ||
		!consumeExecutionJSONString(&remaining, result.Digest) ||
		!consumeExecutionJSONString(&remaining, result.CacheKey) ||
		!consumeRenderSegmentsJSONBudget(&remaining, result.Render, limits) {
		return ErrExecutionLimit
	}
	for _, tag := range result.CacheTags {
		if !consumeExecutionJSONString(&remaining, tag) {
			return ErrExecutionLimit
		}
	}
	for _, attribution := range result.Attribution {
		if !consumeExecutionJSONBudget(&remaining, 192) ||
			!consumeExecutionJSONString(&remaining, attribution.ContentID) ||
			!consumeExecutionJSONString(&remaining, attribution.ContractVersion) ||
			!consumeExecutionJSONString(&remaining, attribution.Action) ||
			!consumeExecutionJSONString(&remaining, attribution.Artifact.ExtensionID) ||
			!consumeExecutionJSONString(&remaining, attribution.Artifact.ExtensionVersion) ||
			!consumeExecutionJSONString(&remaining, attribution.Artifact.PackageDigest) ||
			!consumeExecutionJSONString(&remaining, attribution.Artifact.RuntimeInstanceID) {
			return ErrExecutionLimit
		}
	}
	return nil
}

func escapedHTMLTextSize(value string, limit int) (int, error) {
	size := 0
	for index := 0; index < len(value); index++ {
		addition := 1
		switch value[index] {
		case '&':
			addition = len("&amp;")
		case '\'':
			addition = len("&#39;")
		case '<', '>':
			addition = 4
		case '"':
			addition = len("&#34;")
		}
		if addition > limit-size {
			return 0, fmt.Errorf("%w: escaped text", ErrExecutionLimit)
		}
		size += addition
	}
	return size, nil
}

func appendPlainTextBounded(builder interface {
	Len() int
	WriteByte(byte) error
	WriteString(string) (int, error)
}, value string, limit int) error {
	if value == "" {
		return nil
	}
	separator := 0
	if builder.Len() > 0 {
		separator = 1
	}
	if len(value) > limit-builder.Len()-separator {
		return ErrExecutionLimit
	}
	if separator == 1 {
		if err := builder.WriteByte(' '); err != nil {
			return err
		}
	}
	_, err := builder.WriteString(value)
	return err
}
