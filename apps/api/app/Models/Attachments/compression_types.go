package attachments

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"strconv"
	"strings"
	"time"
)

type CompressionStore interface {
	CreateCompressionTask(context.Context, Attachment, CompressionSettings) (int64, bool, error)
	ClaimCompressionTask(context.Context, int64) (CompressionTask, error)
	CompleteCompressionTask(context.Context, CompressionTask, AttachmentVariant) (*AttachmentVariant, error)
	FinishCompressionTask(context.Context, int64, string, string) error
	GetAttachmentVariant(context.Context, int64, string) (AttachmentVariant, error)
	CompressionStats(context.Context) (CompressionStats, error)
	BackfillCompressionTasks(context.Context, CompressionSettings, int) ([]int64, error)
}

const (
	CompressionVariantDisplay = "display"

	CompressionStatusPending   = "pending"
	CompressionStatusRunning   = "running"
	CompressionStatusSucceeded = "succeeded"
	CompressionStatusSkipped   = "skipped"
	CompressionStatusFailed    = "failed"

	RecommendedCompressionStrength          = 55
	RecommendedCompressionMaxDimension      = 2560
	RecommendedCompressionMinSizeKB         = 256
	RecommendedCompressionMinSavingsPercent = 8
)

type CompressionSettings struct {
	Enabled           bool   `json:"enabled"`
	Strength          int    `json:"strength"`
	MaxDimension      int    `json:"maxDimension"`
	MinSizeKB         int    `json:"minSizeKb"`
	MinSavingsPercent int    `json:"minSavingsPercent"`
	JPEGQuality       int    `json:"jpegQuality"`
	PolicyDigest      string `json:"policyDigest"`
}

func (s CompressionSettings) validInput() bool {
	return s.Strength >= 0 && s.Strength <= 100 &&
		s.MaxDimension >= 320 && s.MaxDimension <= 8192 &&
		s.MinSizeKB >= 1 && s.MinSizeKB <= 1024*1024 &&
		s.MinSavingsPercent >= 0 && s.MinSavingsPercent <= 90
}

func (s CompressionSettings) normalized() CompressionSettings {
	if s.Strength < 0 || s.Strength > 100 {
		s.Strength = RecommendedCompressionStrength
	}
	if s.MaxDimension < 320 || s.MaxDimension > 8192 {
		s.MaxDimension = RecommendedCompressionMaxDimension
	}
	if s.MinSizeKB < 1 {
		s.MinSizeKB = RecommendedCompressionMinSizeKB
	}
	if s.MinSavingsPercent < 0 || s.MinSavingsPercent > 90 {
		s.MinSavingsPercent = RecommendedCompressionMinSavingsPercent
	}
	s.JPEGQuality = compressionJPEGQuality(s.Strength)
	payload := strings.Join([]string{
		strconv.FormatBool(s.Enabled), strconv.Itoa(s.Strength), strconv.Itoa(s.MaxDimension),
		strconv.Itoa(s.MinSizeKB), strconv.Itoa(s.MinSavingsPercent),
	}, "\x00")
	digest := sha256.Sum256([]byte(payload))
	s.PolicyDigest = hex.EncodeToString(digest[:])
	return s
}

func compressionJPEGQuality(strength int) int {
	if strength < 0 {
		strength = 0
	}
	if strength > 100 {
		strength = 100
	}
	return 95 - (strength*25+50)/100
}

type AttachmentVariant struct {
	ID                  int64     `json:"id"`
	AttachmentID        int64     `json:"attachmentId"`
	Name                string    `json:"name"`
	Provider            string    `json:"provider"`
	ObjectKey           string    `json:"objectKey"`
	ContentType         string    `json:"contentType"`
	SizeBytes           int64     `json:"size"`
	SHA256              string    `json:"sha256"`
	ImageWidth          int       `json:"imageWidth"`
	ImageHeight         int       `json:"imageHeight"`
	SourceSHA256        string    `json:"sourceSha256"`
	PolicyDigest        string    `json:"policyDigest"`
	CompressionStrength int       `json:"compressionStrength"`
	CreatedAt           time.Time `json:"createdAt"`
	UpdatedAt           time.Time `json:"updatedAt"`
}

type CompressionTask struct {
	ID                  int64
	Attachment          Attachment
	VariantName         string
	SourceSHA256        string
	PolicyDigest        string
	CompressionStrength int
	Attempts            int
}

type CompressionStats struct {
	Pending       int64 `json:"pending"`
	Running       int64 `json:"running"`
	Failed        int64 `json:"failed"`
	ReadyVariants int64 `json:"readyVariants"`
	OriginalBytes int64 `json:"originalBytes"`
	VariantBytes  int64 `json:"variantBytes"`
	SavedBytes    int64 `json:"savedBytes"`
}

type CompressionBackfillResult struct {
	Scheduled int64 `json:"scheduled"`
}

type VariantContent struct {
	ContentType  string
	OriginalName string
}
