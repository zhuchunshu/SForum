package forum

import (
	"encoding/json"
	"math"
	"net/url"
	"regexp"
	"strings"

	editordocument "github.com/zhuchunshu/sforum/apps/api/app/Support/EditorDocument"
)

var attachmentPublicIDPattern = regexp.MustCompile(`^[a-f0-9]{32}$`)

func normalizeAndValidateContentAttachmentIDs(
	content RenderedContent,
	input *[]int64,
) ([]int64, bool, error) {
	attachmentIDs, submitted, err := normalizeContentAttachmentIDs(input)
	if err != nil {
		return nil, submitted, err
	}
	if err := validateEditorDocumentAttachmentIDs(content, attachmentIDs, submitted); err != nil {
		return nil, submitted, err
	}
	return attachmentIDs, submitted, nil
}

// validateEditorDocumentAttachmentIDs keeps native image identity and the
// transactional attachment reference list consistent. Other attachment kinds
// may still be supplied through attachmentIds by API clients.
func validateEditorDocumentAttachmentIDs(content RenderedContent, attachmentIDs []int64, submitted bool) error {
	if content.SourceFormat != SourceFormatEditorDocument {
		return nil
	}

	var document editordocument.Document
	if err := json.Unmarshal([]byte(content.RawContent), &document); err != nil {
		return ErrInvalidContent
	}

	nodeIDs, err := editorDocumentImageAttachmentIDs(document.Content)
	if err != nil {
		return err
	}
	if len(nodeIDs) == 0 {
		return nil
	}
	if !submitted {
		return ErrInvalidContent
	}

	allowed := make(map[int64]struct{}, len(attachmentIDs))
	for _, id := range attachmentIDs {
		allowed[id] = struct{}{}
	}
	for _, id := range nodeIDs {
		if _, ok := allowed[id]; !ok {
			return ErrInvalidContent
		}
	}
	return nil
}

func editorDocumentImageAttachmentIDs(nodes []editordocument.Node) ([]int64, error) {
	seen := map[int64]struct{}{}
	ids := make([]int64, 0)
	var walk func([]editordocument.Node) error
	walk = func(items []editordocument.Node) error {
		for _, node := range items {
			if node.Type == "image" {
				idValue, hasID := node.Attrs["attachmentId"]
				publicValue, hasPublicID := node.Attrs["attachmentPublicId"]
				if hasID != hasPublicID {
					return ErrInvalidContent
				}
				if hasID {
					id, ok := positiveJSONInt64(idValue)
					publicID, publicOK := publicValue.(string)
					publicID = strings.TrimSpace(publicID)
					src, _ := node.Attrs["src"].(string)
					if !ok || !publicOK || !attachmentPublicIDPattern.MatchString(publicID) || !attachmentURLMatchesPublicID(src, publicID) {
						return ErrInvalidContent
					}
					if _, exists := seen[id]; !exists {
						seen[id] = struct{}{}
						ids = append(ids, id)
					}
				}
			}
			if err := walk(node.Content); err != nil {
				return err
			}
		}
		return nil
	}
	if err := walk(nodes); err != nil {
		return nil, err
	}
	return ids, nil
}

func positiveJSONInt64(value any) (int64, bool) {
	number, ok := value.(float64)
	if !ok || number <= 0 || number > math.MaxInt64 || math.Trunc(number) != number {
		return 0, false
	}
	return int64(number), true
}

func attachmentURLMatchesPublicID(rawURL, publicID string) bool {
	parsed, err := url.Parse(strings.TrimSpace(rawURL))
	if err != nil {
		return false
	}
	switch parsed.Path {
	case "/media/attachments/" + publicID,
		"/api/v1/attachments/" + publicID + "/content",
		"/api/v1/attachments/" + publicID + "/variants/display/content":
		return true
	default:
		return false
	}
}
