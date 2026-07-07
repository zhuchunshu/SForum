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
)

var definitions = []Definition{
	{
		Name:          ExtensionEnabled,
		Kind:          KindObserve,
		Description:   "Emitted after a plugin is enabled and its runtime starts.",
		PayloadFields: []string{"extensionId"},
		TimeoutMS:     DefaultAsyncTimeoutMS,
	},
	{
		Name:          ExtensionDisabled,
		Kind:          KindObserve,
		Description:   "Emitted after a plugin is disabled and its runtime stops.",
		PayloadFields: []string{"extensionId"},
		TimeoutMS:     DefaultAsyncTimeoutMS,
	},
	{
		Name:          UserRegistered,
		Kind:          KindObserve,
		Description:   "Emitted after a new user is committed.",
		PayloadFields: []string{"userId", "username", "email", "locale"},
		TimeoutMS:     DefaultAsyncTimeoutMS,
	},
	{
		Name:          TopicBeforeCreate,
		Kind:          KindFilter,
		Description:   "Runs before a topic is committed and may reject or patch allowlisted input.",
		PayloadFields: []string{"actorUserId", "categorySlug", "tagSlugs", "title", "content"},
		PatchFields:   []string{"categorySlug", "tagSlugs", "title", "content"},
		TimeoutMS:     DefaultSyncTimeoutMS,
	},
	{
		Name:          TopicCreated,
		Kind:          KindObserve,
		Description:   "Emitted after a topic is committed.",
		PayloadFields: []string{"topicId", "authorUserId", "categorySlug", "tagSlugs", "title"},
		TimeoutMS:     DefaultAsyncTimeoutMS,
	},
	{
		Name:          TopicUpdated,
		Kind:          KindObserve,
		Description:   "Emitted after a topic's content or taxonomy is updated.",
		PayloadFields: []string{"topicId", "actorUserId", "title", "categorySlug", "tagSlugs"},
		TimeoutMS:     DefaultAsyncTimeoutMS,
	},
	{
		Name:          TopicDeleted,
		Kind:          KindObserve,
		Description:   "Emitted after a topic is soft-deleted.",
		PayloadFields: []string{"topicId", "actorUserId"},
		TimeoutMS:     DefaultAsyncTimeoutMS,
	},
	{
		Name:          TopicHidden,
		Kind:          KindObserve,
		Description:   "Emitted after a topic is hidden by a moderator.",
		PayloadFields: []string{"topicId", "actorUserId"},
		TimeoutMS:     DefaultAsyncTimeoutMS,
	},
	{
		Name:          TopicRestored,
		Kind:          KindObserve,
		Description:   "Emitted after a hidden or deleted topic is restored to active.",
		PayloadFields: []string{"topicId", "actorUserId"},
		TimeoutMS:     DefaultAsyncTimeoutMS,
	},
	{
		Name:          TopicLocked,
		Kind:          KindObserve,
		Description:   "Emitted after a topic is locked.",
		PayloadFields: []string{"topicId", "actorUserId"},
		TimeoutMS:     DefaultAsyncTimeoutMS,
	},
	{
		Name:          TopicUnlocked,
		Kind:          KindObserve,
		Description:   "Emitted after a topic is unlocked.",
		PayloadFields: []string{"topicId", "actorUserId"},
		TimeoutMS:     DefaultAsyncTimeoutMS,
	},
	{
		Name:          TopicPinned,
		Kind:          KindObserve,
		Description:   "Emitted after a topic is pinned.",
		PayloadFields: []string{"topicId", "actorUserId"},
		TimeoutMS:     DefaultAsyncTimeoutMS,
	},
	{
		Name:          TopicUnpinned,
		Kind:          KindObserve,
		Description:   "Emitted after a topic is unpinned.",
		PayloadFields: []string{"topicId", "actorUserId"},
		TimeoutMS:     DefaultAsyncTimeoutMS,
	},
	{
		Name:          CategoryCreated,
		Kind:          KindObserve,
		Description:   "Emitted after a category is created.",
		PayloadFields: []string{"categoryId", "categorySlug", "groupId"},
		TimeoutMS:     DefaultAsyncTimeoutMS,
	},
	{
		Name:          CategoryUpdated,
		Kind:          KindObserve,
		Description:   "Emitted after a category is updated.",
		PayloadFields: []string{"categoryId", "categorySlug", "groupId"},
		TimeoutMS:     DefaultAsyncTimeoutMS,
	},
	{
		Name:          TagCreated,
		Kind:          KindObserve,
		Description:   "Emitted after a tag is created.",
		PayloadFields: []string{"tagId", "tagSlug", "status"},
		TimeoutMS:     DefaultAsyncTimeoutMS,
	},
	{
		Name:          TagUpdated,
		Kind:          KindObserve,
		Description:   "Emitted after a tag is updated.",
		PayloadFields: []string{"tagId", "tagSlug", "status"},
		TimeoutMS:     DefaultAsyncTimeoutMS,
	},
	{
		Name:          CommentCreated,
		Kind:          KindObserve,
		Description:   "Emitted after a comment is committed.",
		PayloadFields: []string{"commentId", "topicId", "authorUserId", "parentId"},
		TimeoutMS:     DefaultAsyncTimeoutMS,
	},
	{
		Name:          AttachmentUploaded,
		Kind:          KindObserve,
		Description:   "Emitted after attachment metadata is committed.",
		PayloadFields: []string{"attachmentId", "publicId", "ownerUserId", "provider", "contentType", "sizeBytes"},
		TimeoutMS:     DefaultAsyncTimeoutMS,
	},
}

func Definitions() []Definition {
	items := make([]Definition, len(definitions))
	copy(items, definitions)
	for index := range items {
		items[index].PayloadFields = append([]string{}, items[index].PayloadFields...)
		items[index].PatchFields = append([]string{}, items[index].PatchFields...)
	}
	return items
}

func FindDefinition(name string) (Definition, bool) {
	for _, definition := range definitions {
		if definition.Name == name {
			copy := definition
			copy.PayloadFields = append([]string{}, definition.PayloadFields...)
			copy.PatchFields = append([]string{}, definition.PatchFields...)
			return copy, true
		}
	}
	return Definition{}, false
}

func Known(name string) bool {
	_, ok := FindDefinition(name)
	return ok
}
