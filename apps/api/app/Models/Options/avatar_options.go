package options

import (
	"context"
	"strings"
)

// 头像 provider 枚举。default_provider 用其中之一。
const (
	AvatarProviderGravatar = "gravatar"
	AvatarProviderStatic   = "static"
	AvatarProviderInitials = "initials"
)

// Gravatar hash 算法。sha256 是 Gravatar 当前推荐值，md5 仅用于兼容旧镜像源。
const (
	AvatarHashSHA256 = "sha256"
	AvatarHashMD5    = "md5"
)

// 数值边界常量。集中在便于后台 UI 与后端校验保持一致。
const (
	avatarMaxSizeKBDefault       = 2048
	avatarMaxSizeKBMin           = 1
	avatarMaxSizeKBMax           = 10240
	avatarMaxDimensionDefault    = 2048
	avatarTargetDimensionDefault = 256
	avatarDimensionMin           = 32
	avatarDimensionMax           = 4096
	avatarCompressQualityDefault = 85
	avatarCompressQualityMin     = 1
	avatarCompressQualityMax     = 100
)

var avatarProviders = []string{
	AvatarProviderGravatar,
	AvatarProviderStatic,
	AvatarProviderInitials,
}

var avatarHashAlgorithms = []string{
	AvatarHashSHA256,
	AvatarHashMD5,
}

func avatarOptionNames() []string {
	return []string{
		NameAvatarAllowUpload,
		NameAvatarDefaultProvider,
		NameAvatarGravatarBaseURL,
		NameAvatarGravatarHashAlgorithm,
		NameAvatarDefaultStaticURL,
		NameAvatarMaxSizeKB,
		NameAvatarMaxDimension,
		NameAvatarAllowGIF,
		NameAvatarCompressEnabled,
		NameAvatarTargetDimension,
		NameAvatarCompressQuality,
	}
}

// AvatarOptions 是头像运行时配置的强类型视图，供 Profile/头像服务消费。
type AvatarOptions struct {
	AllowUpload           bool
	DefaultProvider       string
	GravatarBaseURL       string
	GravatarHashAlgorithm string
	DefaultStaticURL      string
	MaxSizeKB             int
	MaxDimension          int
	AllowGIF              bool
	CompressEnabled       bool
	TargetDimension       int
	CompressQuality       int
}

// AvatarOptions 读取头像运行时配置，并在某项缺失时回退到推荐默认值。
func (s *Service) AvatarOptions(ctx context.Context) (AvatarOptions, error) {
	values, err := s.loadMap(ctx)
	if err != nil {
		return AvatarOptions{}, err
	}
	return avatarOptionsFromValues(values), nil
}

func avatarOptionsFromValues(values map[string]string) AvatarOptions {
	maxSizeKB, _ := parseBoundedInt(values[NameAvatarMaxSizeKB], avatarMaxSizeKBMin, avatarMaxSizeKBMax)
	if maxSizeKB <= 0 {
		maxSizeKB = avatarMaxSizeKBDefault
	}
	maxDimension, _ := parseBoundedInt(values[NameAvatarMaxDimension], avatarDimensionMin, avatarDimensionMax)
	if maxDimension <= 0 {
		maxDimension = avatarMaxDimensionDefault
	}
	targetDimension, _ := parseBoundedInt(values[NameAvatarTargetDimension], avatarDimensionMin, avatarDimensionMax)
	if targetDimension <= 0 {
		targetDimension = avatarTargetDimensionDefault
	}
	compressQuality, _ := parseBoundedInt(values[NameAvatarCompressQuality], avatarCompressQualityMin, avatarCompressQualityMax)
	if compressQuality <= 0 {
		compressQuality = avatarCompressQualityDefault
	}
	provider, ok := normalizeAvatarProvider(values[NameAvatarDefaultProvider])
	if !ok {
		provider = AvatarProviderInitials
	}
	hashAlgorithm, ok := normalizeAvatarHashAlgorithm(values[NameAvatarGravatarHashAlgorithm])
	if !ok {
		hashAlgorithm = AvatarHashSHA256
	}
	return AvatarOptions{
		AllowUpload:           isEnabledOption(values[NameAvatarAllowUpload]),
		DefaultProvider:       provider,
		GravatarBaseURL:       normalizeGravatarBaseURLValue(values[NameAvatarGravatarBaseURL]),
		GravatarHashAlgorithm: hashAlgorithm,
		DefaultStaticURL:      strings.TrimSpace(values[NameAvatarDefaultStaticURL]),
		MaxSizeKB:             maxSizeKB,
		MaxDimension:          maxDimension,
		AllowGIF:              isEnabledOption(values[NameAvatarAllowGIF]),
		CompressEnabled:       isEnabledOption(values[NameAvatarCompressEnabled]),
		TargetDimension:       targetDimension,
		CompressQuality:       compressQuality,
	}
}

// coerceAvatarOptions 对所有头像选项逐项做归一化，无效值回退到默认。
func coerceAvatarOptions(values map[string]string, defaults map[string]string) {
	for _, name := range avatarOptionNames() {
		normalized, ok := normalizeOptionValue(name, values[name])
		if !ok {
			values[name] = defaults[name]
			continue
		}
		values[name] = normalized
	}
}

func isValidAvatarOptions(values map[string]string) bool {
	for _, name := range avatarOptionNames() {
		if _, ok := normalizeOptionValue(name, values[name]); !ok {
			return false
		}
	}
	// static 类型必须配置默认图 URL；其余 provider 不强制。
	if provider, ok := normalizeAvatarProvider(values[NameAvatarDefaultProvider]); ok && provider == AvatarProviderStatic {
		if _, ok := normalizeOptionalURL(values[NameAvatarDefaultStaticURL]); !ok || strings.TrimSpace(values[NameAvatarDefaultStaticURL]) == "" {
			return false
		}
	}
	return true
}

func normalizeAvatarProvider(value string) (string, bool) {
	value = strings.ToLower(strings.TrimSpace(value))
	if value == "" {
		return AvatarProviderInitials, true
	}
	return normalizeChoice(value, avatarProviders)
}

func normalizeAvatarHashAlgorithm(value string) (string, bool) {
	value = strings.ToLower(strings.TrimSpace(value))
	if value == "" {
		return AvatarHashSHA256, true
	}
	return normalizeChoice(value, avatarHashAlgorithms)
}

// normalizeAvatarGravatarBaseURL 校验 Gravatar 源地址：非空时需为 http/https URL，
// 且以 / 结尾（拼接 hash 时语义更清晰）。空值合法，运行时会用推荐默认填充。
func normalizeAvatarGravatarBaseURL(value string) (string, bool) {
	value = strings.TrimSpace(value)
	if value == "" {
		return "", true
	}
	if !isValidURL(value) {
		return "", false
	}
	return strings.TrimRight(value, "/") + "/", true
}

// normalizeGravatarBaseURLValue 保证运行时始终拿到可用的基址（空则用推荐默认）。
func normalizeGravatarBaseURLValue(value string) string {
	normalized, ok := normalizeAvatarGravatarBaseURL(value)
	if !ok || normalized == "" {
		return "https://gravatar.com/avatar/"
	}
	return normalized
}
