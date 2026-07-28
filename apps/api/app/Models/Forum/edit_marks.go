package forum

func applyTopicEditMark(topic TopicDetail, show bool) TopicDetail {
	topic.Edited = show && (topic.ContentEdited || topic.EditedAt != nil)
	if !topic.Edited {
		topic.EditedAt = nil
	}
	return topic
}

func applyCommentEditMarks(items []Comment, show bool) []Comment {
	for i := range items {
		items[i].Edited = show && (items[i].ContentEdited || items[i].EditedAt != nil)
		if !items[i].Edited {
			items[i].EditedAt = nil
		}
		if len(items[i].Children) > 0 {
			items[i].Children = applyCommentEditMarks(items[i].Children, show)
		}
	}
	return items
}

func applyCommentEditMark(comment Comment, show bool) Comment {
	marked := applyCommentEditMarks([]Comment{comment}, show)
	return marked[0]
}
