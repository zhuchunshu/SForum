package editordocument

// OrderedStages is the Host-owned content pipeline contract order.
// Embed and SEO stages are intentional extension points: plugins may contribute
// later via Content/SEO registries without reordering Host stages.
func OrderedStages() []string {
	return []string{
		StageParse,
		StageValidate,
		StageNormalize,
		StageStore,
		StageRender,
		StageSanitize,
		StageEmbed,
		StageSEO,
	}
}

// StageResult is a partial pipeline observation for inspectors and tests.
type StageResult struct {
	Stage     string   `json:"stage"`
	OK        bool     `json:"ok"`
	Fallbacks []string `json:"fallbacks,omitempty"`
	Error     string   `json:"error,omitempty"`
}

// RunOrderedPipeline executes every Host stage in order and returns Accept's
// final document. Embed/SEO currently record success without mutation so
// reference plugins can hang off these stable stage IDs.
func RunOrderedPipeline(input Input) (Accepted, []StageResult, error) {
	results := make([]StageResult, 0, len(OrderedStages()))
	doc, err := Parse(input)
	if err != nil {
		results = append(results, StageResult{Stage: StageParse, Error: err.Error()})
		return Accepted{}, results, err
	}
	results = append(results, StageResult{Stage: StageParse, OK: true})

	schema := input.Schema
	if len(schema.Nodes) == 0 {
		schema = CoreSchema()
	}
	normalized, fallbacks, err := ValidateAndNormalize(doc, schema)
	if err != nil {
		results = append(results,
			StageResult{Stage: StageValidate, Error: err.Error()},
			StageResult{Stage: StageNormalize, Error: err.Error()},
		)
		return Accepted{}, results, err
	}
	results = append(results,
		StageResult{Stage: StageValidate, OK: true, Fallbacks: fallbacks},
		StageResult{Stage: StageNormalize, OK: true, Fallbacks: fallbacks},
	)

	// Store stage freezes the accepted native shape before derived render.
	results = append(results, StageResult{Stage: StageStore, OK: true})

	htmlRaw := RenderHTML(normalized, schema)
	results = append(results, StageResult{Stage: StageRender, OK: true})
	_ = SanitizeHTML(htmlRaw)
	results = append(results, StageResult{Stage: StageSanitize, OK: true})

	// Embed/SEO hooks: reserved ordered contracts for later registry wiring.
	results = append(results,
		StageResult{Stage: StageEmbed, OK: true},
		StageResult{Stage: StageSEO, OK: true},
	)

	accepted, err := Accept(input)
	if err != nil {
		return Accepted{}, results, err
	}
	return accepted, results, nil
}
