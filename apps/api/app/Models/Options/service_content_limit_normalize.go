package options

func coerceForumContentLimitOptions(coerced map[string]string, defaults map[string]string) {
	type bound struct {
		name string
		min  int
		max  int
	}
	for _, item := range []bound{
		{NameForumTopicTitleMinRunes, forumTitleMinRunesMin, forumTitleMinRunesMax},
		{NameForumTopicTitleMaxRunes, forumTitleMaxRunesMin, forumTitleMaxRunesMax},
		{NameForumTopicContentMinRunes, forumContentMinRunesMin, forumContentMinRunesMax},
		{NameForumTopicContentMaxRunes, forumContentMaxRunesMin, forumContentMaxRunesMax},
		{NameForumCommentMinRunes, forumCommentMinRunesMin, forumCommentMinRunesMax},
		{NameForumCommentMaxRunes, forumCommentMaxRunesMin, forumCommentMaxRunesMax},
		{NameForumCommentMaxNestingDepth, forumNestingMin, forumNestingMax},
		{NameForumCommentsTreeDescendantsPerRoot, forumTreeDescendantsMin, forumTreeDescendantsMax},
		{NameForumTopicEditWindowMinutes, forumEditWindowMin, forumEditWindowMax},
		{NameForumCommentEditWindowMinutes, forumEditWindowMin, forumEditWindowMax},
		{NameForumTopicCooldownSeconds, forumCooldownMin, forumCooldownMax},
		{NameForumCommentCooldownSeconds, forumCooldownMin, forumCooldownMax},
		{NameForumDailyTopicLimit, forumDailyLimitMin, forumDailyLimitMax},
		{NameForumDailyCommentLimit, forumDailyLimitMin, forumDailyLimitMax},
		{NameForumExcerptRuneLimit, forumExcerptMin, forumExcerptMax},
	} {
		if _, ok := parseBoundedInt(coerced[item.name], item.min, item.max); !ok {
			coerced[item.name] = defaults[item.name]
		}
	}
	// 标题/正文/评论 min 不得超过 max，否则回退整对。
	resetPair := func(minName, maxName string) {
		minVal, okMin := parseBoundedInt(coerced[minName], 0, 1<<30)
		maxVal, okMax := parseBoundedInt(coerced[maxName], 0, 1<<30)
		if okMin && okMax && minVal > maxVal {
			coerced[minName] = defaults[minName]
			coerced[maxName] = defaults[maxName]
		}
	}
	resetPair(NameForumTopicTitleMinRunes, NameForumTopicTitleMaxRunes)
	resetPair(NameForumTopicContentMinRunes, NameForumTopicContentMaxRunes)
	resetPair(NameForumCommentMinRunes, NameForumCommentMaxRunes)
}

func validForumContentLimitOptionValues(values map[string]string) bool {
	type bound struct {
		name string
		min  int
		max  int
	}
	for _, item := range []bound{
		{NameForumTopicTitleMinRunes, forumTitleMinRunesMin, forumTitleMinRunesMax},
		{NameForumTopicTitleMaxRunes, forumTitleMaxRunesMin, forumTitleMaxRunesMax},
		{NameForumTopicContentMinRunes, forumContentMinRunesMin, forumContentMinRunesMax},
		{NameForumTopicContentMaxRunes, forumContentMaxRunesMin, forumContentMaxRunesMax},
		{NameForumCommentMinRunes, forumCommentMinRunesMin, forumCommentMinRunesMax},
		{NameForumCommentMaxRunes, forumCommentMaxRunesMin, forumCommentMaxRunesMax},
		{NameForumCommentMaxNestingDepth, forumNestingMin, forumNestingMax},
		{NameForumCommentsTreeDescendantsPerRoot, forumTreeDescendantsMin, forumTreeDescendantsMax},
		{NameForumTopicEditWindowMinutes, forumEditWindowMin, forumEditWindowMax},
		{NameForumCommentEditWindowMinutes, forumEditWindowMin, forumEditWindowMax},
		{NameForumTopicCooldownSeconds, forumCooldownMin, forumCooldownMax},
		{NameForumCommentCooldownSeconds, forumCooldownMin, forumCooldownMax},
		{NameForumDailyTopicLimit, forumDailyLimitMin, forumDailyLimitMax},
		{NameForumDailyCommentLimit, forumDailyLimitMin, forumDailyLimitMax},
		{NameForumExcerptRuneLimit, forumExcerptMin, forumExcerptMax},
	} {
		if _, ok := parseBoundedInt(values[item.name], item.min, item.max); !ok {
			return false
		}
	}
	pairOK := func(minName, maxName string) bool {
		minVal, okMin := parseBoundedInt(values[minName], 0, 1<<30)
		maxVal, okMax := parseBoundedInt(values[maxName], 0, 1<<30)
		return okMin && okMax && minVal <= maxVal
	}
	return pairOK(NameForumTopicTitleMinRunes, NameForumTopicTitleMaxRunes) &&
		pairOK(NameForumTopicContentMinRunes, NameForumTopicContentMaxRunes) &&
		pairOK(NameForumCommentMinRunes, NameForumCommentMaxRunes)
}
