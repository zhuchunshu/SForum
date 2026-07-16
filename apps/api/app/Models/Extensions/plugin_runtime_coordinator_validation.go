package extensions

import (
	"strings"
	"unicode/utf8"
)

func validPluginRuntimeCoordinatorNode(node PluginRuntimeNode, identity PluginRuntimeNodeIdentity) bool {
	return node.PluginRuntimeNodeIdentity == identity && node.LastAppliedRevision >= 0 &&
		!node.FirstSeenAt.IsZero() && !node.LastSeenAt.IsZero() && !node.LeaseExpiresAt.IsZero() &&
		!node.LastSeenAt.Before(node.FirstSeenAt) && node.LeaseExpiresAt.After(node.LastSeenAt)
}

func validPluginRuntimeApplyingAck(
	ack PluginRuntimePublicationAck,
	identity PluginRuntimeNodeIdentity,
	publicationRevision int64,
) bool {
	return ack.PluginRuntimeNodeIdentity == identity && ack.PublicationRevision == publicationRevision &&
		ack.Status == PluginRuntimeAckApplying && ack.AppliedMemberCount == nil &&
		ack.AppliedMembersDigest == "" && ack.ErrorReason == "" && ack.AppliedAt == nil &&
		ack.AttemptCount > 0 && ack.Revision > 0 && validPluginRuntimeAckTimes(ack) &&
		ack.StartedAt.Equal(ack.UpdatedAt)
}

func validPluginRuntimeCompletedAck(
	completed PluginRuntimePublicationAck,
	started PluginRuntimePublicationAck,
	publication PluginRuntimePublication,
) bool {
	return completed.PluginRuntimeNodeIdentity == started.PluginRuntimeNodeIdentity &&
		completed.PublicationRevision == publication.Revision && completed.Status == PluginRuntimeAckApplied &&
		completed.AppliedMemberCount != nil && *completed.AppliedMemberCount == publication.MemberCount &&
		completed.AppliedMembersDigest == publication.MembersDigest && completed.ErrorReason == "" &&
		completed.AppliedAt != nil && completed.AttemptCount == started.AttemptCount &&
		completed.Revision == started.Revision+1 && completed.StartedAt.Equal(started.StartedAt) &&
		!completed.UpdatedAt.Before(started.UpdatedAt) && validPluginRuntimeAckTimes(completed) &&
		!completed.AppliedAt.Before(completed.StartedAt) && !completed.AppliedAt.After(completed.UpdatedAt)
}

func validPluginRuntimeFailedAck(
	failed PluginRuntimePublicationAck,
	started PluginRuntimePublicationAck,
	publicationRevision int64,
) bool {
	return failed.PluginRuntimeNodeIdentity == started.PluginRuntimeNodeIdentity &&
		failed.PublicationRevision == publicationRevision && failed.Status == PluginRuntimeAckFailed &&
		failed.AppliedMemberCount == nil && failed.AppliedMembersDigest == "" &&
		strings.TrimSpace(failed.ErrorReason) != "" && failed.AppliedAt == nil &&
		failed.AttemptCount == started.AttemptCount && failed.Revision == started.Revision+1 &&
		failed.StartedAt.Equal(started.StartedAt) && !failed.UpdatedAt.Before(started.UpdatedAt) &&
		validPluginRuntimeAckTimes(failed)
}

func validPluginRuntimeAckTimes(ack PluginRuntimePublicationAck) bool {
	return !ack.StartedAt.IsZero() && !ack.UpdatedAt.IsZero() && !ack.UpdatedAt.Before(ack.StartedAt)
}

func clonePluginRuntimePublication(publication PluginRuntimePublication) PluginRuntimePublication {
	publication.Members = append([]PluginRuntimeMember(nil), publication.Members...)
	return publication
}

func pluginRuntimeCoordinatorFailureReason(err error) string {
	reason := "plugin runtime apply failed"
	if err != nil {
		reason = strings.TrimSpace(strings.ToValidUTF8(err.Error(), "\uFFFD"))
		if reason == "" {
			reason = "plugin runtime apply failed"
		}
	}
	for len([]byte(reason)) > 2048 {
		_, size := utf8.DecodeLastRuneInString(reason)
		if size <= 0 {
			return "plugin runtime apply failed"
		}
		reason = reason[:len(reason)-size]
	}
	return reason
}

func receivePluginRuntimeHeartbeatError(errorsCh <-chan error) error {
	select {
	case err := <-errorsCh:
		return err
	default:
		return nil
	}
}
