package events

const (
	ExtensionEnabled   = "extension.enabled"
	ExtensionDisabled  = "extension.disabled"
	UserRegistered     = "user.registered"
	TopicBeforeCreate  = "topic.before_create"
	TopicCreated       = "topic.created"
	TopicUpdated       = "topic.updated"
	TopicDeleted       = "topic.deleted"
	TopicHidden        = "topic.hidden"
	TopicRestored      = "topic.restored"
	TopicLocked        = "topic.locked"
	TopicUnlocked      = "topic.unlocked"
	TopicPinned        = "topic.pinned"
	TopicUnpinned      = "topic.unpinned"
	CommentCreated     = "comment.created"
	CategoryCreated    = "category.created"
	CategoryUpdated    = "category.updated"
	TagCreated         = "tag.created"
	TagUpdated         = "tag.updated"
	AttachmentUploaded = "attachment.uploaded"
	// EntityMetaUpdated F4.4：实体自定义字段值变更后发出（observe，不阻断写路径）。
	EntityMetaUpdated = "entity_meta.updated"
)

// 目录约定（F1.3）：
//   - observe：异步投递，TimeoutMS 为 worker 内单次 invoke 上限；失败不阻断业务写路径。
//   - filter/validate：同步调用，TimeoutMS 由 host context 强制；FailurePolicy 默认 fail_closed。
//   - 禁止在 filter 内做重 I/O、批量索引、发信；应 enqueue job。
var definitions = []Definition{
	observe(ExtensionEnabled, "Emitted after a plugin is enabled and its runtime starts.", []string{"extensionId"}),
	observe(ExtensionDisabled, "Emitted after a plugin is disabled and its runtime stops.", []string{"extensionId"}),
	observe(UserRegistered, "Emitted after a new user is committed.", []string{"userId", "username", "email", "locale"}),
	filter(TopicBeforeCreate,
		"Runs before a topic is committed and may reject or patch allowlisted input. Heavy work must enqueue jobs, never block this filter.",
		[]string{"actorUserId", "categorySlug", "tagSlugs", "title", "content"},
		[]string{"categorySlug", "tagSlugs", "title", "content"},
	),
	observe(TopicCreated, "Emitted after a topic is committed.", []string{"topicId", "authorUserId", "categorySlug", "tagSlugs", "title"}),
	observe(TopicUpdated, "Emitted after a topic's content or taxonomy is updated.", []string{"topicId", "actorUserId", "title", "categorySlug", "tagSlugs"}),
	observe(TopicDeleted, "Emitted after a topic is soft-deleted.", []string{"topicId", "actorUserId"}),
	observe(TopicHidden, "Emitted after a topic is hidden by a moderator.", []string{"topicId", "actorUserId"}),
	observe(TopicRestored, "Emitted after a hidden or deleted topic is restored to active.", []string{"topicId", "actorUserId"}),
	observe(TopicLocked, "Emitted after a topic is locked.", []string{"topicId", "actorUserId"}),
	observe(TopicUnlocked, "Emitted after a topic is unlocked.", []string{"topicId", "actorUserId"}),
	observe(TopicPinned, "Emitted after a topic is pinned.", []string{"topicId", "actorUserId"}),
	observe(TopicUnpinned, "Emitted after a topic is unpinned.", []string{"topicId", "actorUserId"}),
	observe(CategoryCreated, "Emitted after a category is created.", []string{"categoryId", "categorySlug", "groupId"}),
	observe(CategoryUpdated, "Emitted after a category is updated.", []string{"categoryId", "categorySlug", "groupId"}),
	observe(TagCreated, "Emitted after a tag is created.", []string{"tagId", "tagSlug", "status"}),
	observe(TagUpdated, "Emitted after a tag is updated.", []string{"tagId", "tagSlug", "status"}),
	observe(CommentCreated, "Emitted after a comment is committed.", []string{"commentId", "topicId", "authorUserId", "parentId"}),
	observe(AttachmentUploaded, "Emitted after attachment metadata is committed.", []string{"attachmentId", "publicId", "ownerUserId", "provider", "contentType", "sizeBytes"}),
	observe(EntityMetaUpdated, "Emitted after entity custom field values are written or cleared.", []string{"entityType", "entityId", "fieldKeys", "actorUserId"}),
}

func observe(name, description string, payload []string) Definition {
	return Definition{
		Name:          name,
		Kind:          KindObserve,
		Description:   description,
		PayloadFields: payload,
		TimeoutMS:     DefaultAsyncTimeoutMS,
		// observe 不阻塞写路径；字段保留便于目录 UI 统一展示。
		FailurePolicy: FailurePolicyFailOpen,
	}
}

func filter(name, description string, payload, patch []string) Definition {
	return Definition{
		Name:          name,
		Kind:          KindFilter,
		Description:   description,
		PayloadFields: payload,
		PatchFields:   patch,
		TimeoutMS:     DefaultSyncTimeoutMS,
		FailurePolicy: FailurePolicyFailClosed,
	}
}

func Definitions() []Definition {
	items := make([]Definition, len(definitions))
	copy(items, definitions)
	for index := range items {
		items[index] = normalizeDefinition(items[index])
	}
	return items
}

func FindDefinition(name string) (Definition, bool) {
	for _, definition := range definitions {
		if definition.Name == name {
			return normalizeDefinition(definition), true
		}
	}
	return Definition{}, false
}

func normalizeDefinition(definition Definition) Definition {
	definition.PayloadFields = append([]string{}, definition.PayloadFields...)
	definition.PatchFields = append([]string{}, definition.PatchFields...)
	if definition.TimeoutMS <= 0 {
		if definition.Kind == KindFilter || definition.Kind == KindValidate {
			definition.TimeoutMS = DefaultSyncTimeoutMS
		} else {
			definition.TimeoutMS = DefaultAsyncTimeoutMS
		}
	}
	if definition.FailurePolicy == "" {
		if definition.Kind == KindFilter || definition.Kind == KindValidate {
			definition.FailurePolicy = FailurePolicyFailClosed
		} else {
			definition.FailurePolicy = FailurePolicyFailOpen
		}
	}
	return definition
}

func Known(name string) bool {
	_, ok := FindDefinition(name)
	return ok
}
