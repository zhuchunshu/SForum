package events

const (
	ExtensionEnabled   = "extension.enabled"
	ExtensionDisabled  = "extension.disabled"
	UserRegistered     = "user.registered"
	TopicBeforeCreate  = "topic.before_create"
	TopicCreated       = "topic.created"
	CommentCreated     = "comment.created"
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
		PayloadFields: []string{"actorUserId", "categorySlug", "title", "content"},
		PatchFields:   []string{"categorySlug", "title", "content"},
		TimeoutMS:     DefaultSyncTimeoutMS,
	},
	{
		Name:          TopicCreated,
		Kind:          KindObserve,
		Description:   "Emitted after a topic is committed.",
		PayloadFields: []string{"topicId", "authorUserId", "categorySlug", "title"},
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
