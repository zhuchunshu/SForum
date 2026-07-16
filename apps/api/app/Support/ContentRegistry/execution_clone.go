package contentregistry

import "slices"

func cloneEditorDocument(value EditorDocument) EditorDocument {
	value.Value = slices.Clone(value.Value)
	return value
}

func cloneSerializedContent(value SerializedContent) SerializedContent {
	value.Data = slices.Clone(value.Data)
	return value
}

func cloneRenderSegments(value RenderSegments) RenderSegments {
	value.Segments = slices.Clone(value.Segments)
	return value
}

func cloneExecutionBinding(value ExecutionBinding) ExecutionBinding {
	value.CacheTags = slices.Clone(value.CacheTags)
	return value
}

func cloneExecutionResult(value ExecutionResult) ExecutionResult {
	value.Render = cloneRenderSegments(value.Render)
	value.CacheTags = slices.Clone(value.CacheTags)
	value.Attribution = slices.Clone(value.Attribution)
	return value
}
