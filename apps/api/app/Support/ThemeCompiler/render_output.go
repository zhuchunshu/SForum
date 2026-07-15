package themecompiler

import "strings"

// RenderedHTML is one complete, balanced top-level HTML fragment. Host island
// positions are represented by generated data-sforum-island placeholders; the
// matching typed descriptors are returned separately by RenderOutput.Islands.
type RenderedHTML struct {
	value string
}

func (h RenderedHTML) String() string { return h.value }

type IslandProp struct {
	Name         string         `json:"name"`
	Type         IslandPropType `json:"type"`
	StringValue  string         `json:"stringValue,omitempty"`
	BooleanValue bool           `json:"booleanValue,omitempty"`
	IntegerValue int64          `json:"integerValue,omitempty"`
}

type IslandDescriptor struct {
	// ID is also the value of the generated data-sforum-island placeholder.
	ID                   string       `json:"id"`
	ComponentID          string       `json:"componentId"`
	Props                []IslandProp `json:"props,omitempty"`
	FallbackHTMLSegments []string     `json:"fallbackHtmlSegments,omitempty"`
}

type RenderOutput struct {
	htmlSegments []RenderedHTML
	islands      []IslandDescriptor
	seo          PageSEOView
}

func newRenderOutput(segments []RenderedHTML, islands []IslandDescriptor, seo PageSEOView) RenderOutput {
	return RenderOutput{
		htmlSegments: cloneRenderedHTML(segments),
		islands:      cloneIslandDescriptors(islands),
		seo:          cloneSEOView(seo),
	}
}

// HTMLSegments returns balanced top-level fragments in source order. A
// consumer must replace each data-sforum-island placeholder with the matching
// descriptor during SSR; rendering segments in separate wrapper elements is
// not part of this contract.
func (o RenderOutput) HTMLSegments() []RenderedHTML {
	return cloneRenderedHTML(o.htmlSegments)
}

func (o RenderOutput) Islands() []IslandDescriptor {
	return cloneIslandDescriptors(o.islands)
}

func (o RenderOutput) SEO() PageSEOView {
	return cloneSEOView(o.seo)
}

func cloneRenderedHTML(input []RenderedHTML) []RenderedHTML {
	result := make([]RenderedHTML, len(input))
	for index, segment := range input {
		result[index] = RenderedHTML{value: strings.Clone(segment.value)}
	}
	return result
}

func cloneIslandDescriptors(input []IslandDescriptor) []IslandDescriptor {
	result := make([]IslandDescriptor, len(input))
	for index, descriptor := range input {
		result[index] = descriptor
		result[index].Props = append([]IslandProp(nil), descriptor.Props...)
		result[index].FallbackHTMLSegments = make([]string, len(descriptor.FallbackHTMLSegments))
		for segmentIndex, segment := range descriptor.FallbackHTMLSegments {
			result[index].FallbackHTMLSegments[segmentIndex] = strings.Clone(segment)
		}
	}
	return result
}

func cloneSEOView(input PageSEOView) PageSEOView {
	input.AlternateLinks = append([]AlternateLink(nil), input.AlternateLinks...)
	input.StructuredData = append([]StructuredDataView(nil), input.StructuredData...)
	return input
}
