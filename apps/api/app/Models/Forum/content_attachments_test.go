package forum

import "testing"

func TestValidateEditorDocumentAttachmentIDs(t *testing.T) {
	content := RenderedContent{
		SourceFormat: SourceFormatEditorDocument,
		RawContent:   `{"type":"doc","content":[{"type":"image","attrs":{"src":"/media/attachments/0123456789abcdef0123456789abcdef","attachmentId":42,"attachmentPublicId":"0123456789abcdef0123456789abcdef"}}]}`,
	}

	if err := validateEditorDocumentAttachmentIDs(content, []int64{42}, true); err != nil {
		t.Fatalf("valid image attachment rejected: %v", err)
	}
	if err := validateEditorDocumentAttachmentIDs(content, nil, false); err != ErrInvalidContent {
		t.Fatalf("missing attachment list error=%v", err)
	}
	if err := validateEditorDocumentAttachmentIDs(content, []int64{99}, true); err != ErrInvalidContent {
		t.Fatalf("mismatched attachment list error=%v", err)
	}
}

func TestValidateEditorDocumentAttachmentIdentity(t *testing.T) {
	tests := []string{
		`{"type":"doc","content":[{"type":"image","attrs":{"src":"/api/v1/attachments/0123456789abcdef0123456789abcdef/content","attachmentId":42}}]}`,
		`{"type":"doc","content":[{"type":"image","attrs":{"src":"/api/v1/attachments/ffffffffffffffffffffffffffffffff/content","attachmentId":42,"attachmentPublicId":"0123456789abcdef0123456789abcdef"}}]}`,
		`{"type":"doc","content":[{"type":"image","attrs":{"src":"/api/v1/attachments/0123456789abcdef0123456789abcdef/content","attachmentId":1.5,"attachmentPublicId":"0123456789abcdef0123456789abcdef"}}]}`,
	}
	for _, raw := range tests {
		content := RenderedContent{SourceFormat: SourceFormatEditorDocument, RawContent: raw}
		if err := validateEditorDocumentAttachmentIDs(content, []int64{42}, true); err != ErrInvalidContent {
			t.Fatalf("invalid identity accepted (%s): %v", raw, err)
		}
	}
}

func TestValidateEditorDocumentAllowsExternalImagesAndOrdinaryAttachments(t *testing.T) {
	content := RenderedContent{
		SourceFormat: SourceFormatEditorDocument,
		RawContent:   `{"type":"doc","content":[{"type":"image","attrs":{"src":"https://example.test/image.png","alt":"external"}}]}`,
	}
	if err := validateEditorDocumentAttachmentIDs(content, []int64{7}, true); err != nil {
		t.Fatalf("external image or ordinary attachment rejected: %v", err)
	}
}
