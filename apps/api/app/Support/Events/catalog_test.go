package events

import "testing"

func TestCatalogDocumentsTimeoutAndFailurePolicy(t *testing.T) {
	defs := Definitions()
	if len(defs) == 0 {
		t.Fatal("expected definitions")
	}
	var foundFilter bool
	for _, def := range defs {
		if def.TimeoutMS <= 0 {
			t.Fatalf("%s missing timeoutMs", def.Name)
		}
		if def.FailurePolicy == "" {
			t.Fatalf("%s missing failurePolicy", def.Name)
		}
		switch def.Kind {
		case KindFilter, KindValidate:
			foundFilter = true
			if def.FailurePolicy != FailurePolicyFailClosed {
				t.Fatalf("%s expected fail_closed, got %s", def.Name, def.FailurePolicy)
			}
			if def.TimeoutMS != DefaultSyncTimeoutMS {
				t.Fatalf("%s sync timeout=%d", def.Name, def.TimeoutMS)
			}
		case KindObserve:
			if def.TimeoutMS != DefaultAsyncTimeoutMS {
				t.Fatalf("%s async timeout=%d", def.Name, def.TimeoutMS)
			}
		}
	}
	if !foundFilter {
		t.Fatal("expected at least one filter definition")
	}

	before, ok := FindDefinition(TopicBeforeCreate)
	if !ok {
		t.Fatal("topic.before_create missing")
	}
	if before.TimeoutMS != DefaultSyncTimeoutMS || before.FailurePolicy != FailurePolicyFailClosed {
		t.Fatalf("topic.before_create: %#v", before)
	}

	commentBefore, ok := FindDefinition(CommentBeforeCreate)
	if !ok {
		t.Fatal("comment.before_create missing")
	}
	if commentBefore.Kind != KindFilter {
		t.Fatalf("comment.before_create kind=%s", commentBefore.Kind)
	}
	if commentBefore.TimeoutMS != DefaultSyncTimeoutMS || commentBefore.FailurePolicy != FailurePolicyFailClosed {
		t.Fatalf("comment.before_create: %#v", commentBefore)
	}
	if len(commentBefore.PatchFields) != 1 || commentBefore.PatchFields[0] != "content" {
		t.Fatalf("comment.before_create patch allowlist: %#v", commentBefore.PatchFields)
	}

	topicBeforeUpdate, ok := FindDefinition(TopicBeforeUpdate)
	if !ok {
		t.Fatal("topic.before_update missing")
	}
	if topicBeforeUpdate.Kind != KindFilter {
		t.Fatalf("topic.before_update kind=%s", topicBeforeUpdate.Kind)
	}
	if topicBeforeUpdate.TimeoutMS != DefaultSyncTimeoutMS || topicBeforeUpdate.FailurePolicy != FailurePolicyFailClosed {
		t.Fatalf("topic.before_update: %#v", topicBeforeUpdate)
	}
	wantPatch := []string{"categorySlug", "tagSlugs", "title", "content"}
	if len(topicBeforeUpdate.PatchFields) != len(wantPatch) {
		t.Fatalf("topic.before_update patch allowlist: %#v", topicBeforeUpdate.PatchFields)
	}
	for i, field := range wantPatch {
		if topicBeforeUpdate.PatchFields[i] != field {
			t.Fatalf("topic.before_update patch[%d]=%q want %q", i, topicBeforeUpdate.PatchFields[i], field)
		}
	}

	userBefore, ok := FindDefinition(UserBeforeRegister)
	if !ok {
		t.Fatal("user.before_register missing")
	}
	if userBefore.Kind != KindValidate {
		t.Fatalf("user.before_register kind=%s want validate", userBefore.Kind)
	}
	if userBefore.TimeoutMS != DefaultSyncTimeoutMS || userBefore.FailurePolicy != FailurePolicyFailClosed {
		t.Fatalf("user.before_register: %#v", userBefore)
	}
	if len(userBefore.PatchFields) != 0 {
		t.Fatalf("user.before_register must be reject-only, patch=%#v", userBefore.PatchFields)
	}

	attachmentBefore, ok := FindDefinition(AttachmentBeforeUpload)
	if !ok {
		t.Fatal("attachment.before_upload missing")
	}
	if attachmentBefore.Kind != KindValidate {
		t.Fatalf("attachment.before_upload kind=%s want validate", attachmentBefore.Kind)
	}
	if attachmentBefore.TimeoutMS != DefaultSyncTimeoutMS || attachmentBefore.FailurePolicy != FailurePolicyFailClosed {
		t.Fatalf("attachment.before_upload: %#v", attachmentBefore)
	}
	if len(attachmentBefore.PatchFields) != 0 {
		t.Fatalf("attachment.before_upload must be reject-only, patch=%#v", attachmentBefore.PatchFields)
	}
}
