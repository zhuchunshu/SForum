package extensionscontroller

import (
	"errors"
	"strconv"
	"strings"

	"github.com/gofiber/fiber/v3"

	apphttp "github.com/zhuchunshu/sforum/apps/api/app/Http"
	extensionsruntime "github.com/zhuchunshu/sforum/apps/api/app/Support/Extensions"
	navigationregistry "github.com/zhuchunshu/sforum/apps/api/app/Support/NavigationRegistry"
)

const (
	compositionInspectorDefaultLimit = 50
	compositionInspectorMaximumLimit = 200
	compositionInspectorUnavailable  = "extensions.composition_inspector_unavailable"
	compositionInspectorInvalid      = "extensions.composition_inspector_invalid"
	navigationInspectorUnavailable   = "extensions.navigation_inspector_unavailable"
	navigationInspectorInvalid       = "extensions.navigation_inspector_invalid"
)

var errInspectorLimit = errors.New("inspector limit is invalid")

// ComponentCompositionInspectorSnapshot is a redacted admin view of the live
// composition registry and recent Host-owned traces. Package paths and raw
// props/results are never included.
type ComponentCompositionInspectorSnapshot struct {
	Revision          uint64                                        `json:"revision"`
	SafeMode          bool                                          `json:"safeMode"`
	TargetCount       int                                           `json:"targetCount"`
	ContributionCount int                                           `json:"contributionCount"`
	Conflicts         []extensionsruntime.ComponentProviderConflict `json:"conflicts"`
	Traces            []extensionsruntime.ComponentCompositionTrace `json:"traces"`
}

// NavigationInspectorSnapshot is a redacted admin view of navigation/region
// targets plus recent composition traces.
type NavigationInspectorSnapshot struct {
	Revision          uint64                           `json:"revision"`
	Digest            string                           `json:"digest"`
	SafeMode          bool                             `json:"safeMode"`
	NavigationCount   int                              `json:"navigationCount"`
	RegionCount       int                              `json:"regionCount"`
	ProviderConflicts int                              `json:"providerConflicts"`
	Traces            []navigationregistry.TraceRecord `json:"traces"`
}

func (h *Controller) inspectComponentComposition(c fiber.Ctx) error {
	if _, err := h.routeProviderViewer(c); err != nil {
		return err
	}
	if h.componentComposition == nil || h.componentRegistry == nil {
		return fiber.NewError(fiber.StatusServiceUnavailable, compositionInspectorUnavailable)
	}
	limit, err := parseInspectorLimit(c.Query("limit"), compositionInspectorDefaultLimit, compositionInspectorMaximumLimit)
	if err != nil {
		return fiber.NewError(fiber.StatusUnprocessableEntity, compositionInspectorInvalid)
	}
	snapshot := h.componentRegistry.Snapshot()
	traces := h.componentComposition.InspectorTraces()
	if limit > 0 && len(traces) > limit {
		traces = traces[len(traces)-limit:]
	}
	// 始终返回 [] 而非 null，避免前端 Array.isArray 校验把空注册表当成非法响应。
	conflicts := append([]extensionsruntime.ComponentProviderConflict{}, snapshot.Conflicts...)
	traceCopy := append([]extensionsruntime.ComponentCompositionTrace{}, traces...)
	return apphttp.OK(c, ComponentCompositionInspectorSnapshot{
		Revision:          snapshot.Revision,
		SafeMode:          snapshot.SafeMode,
		TargetCount:       len(snapshot.Targets),
		ContributionCount: len(snapshot.Contributions),
		Conflicts:         conflicts,
		Traces:            traceCopy,
	})
}

func (h *Controller) inspectNavigation(c fiber.Ctx) error {
	if _, err := h.routeProviderViewer(c); err != nil {
		return err
	}
	if h.navigationInspector == nil {
		return fiber.NewError(fiber.StatusServiceUnavailable, navigationInspectorUnavailable)
	}
	limit, err := parseInspectorLimit(c.Query("limit"), compositionInspectorDefaultLimit, compositionInspectorMaximumLimit)
	if err != nil {
		return fiber.NewError(fiber.StatusUnprocessableEntity, navigationInspectorInvalid)
	}
	// 空 target 返回全量摘要 + 最近 traces；授权在 routeProviderViewer。
	inspection, err := h.navigationInspector.Inspect("", limit)
	if err != nil {
		return fiber.NewError(fiber.StatusServiceUnavailable, navigationInspectorUnavailable)
	}
	// 空 traces 也返回 JSON 数组，保持与 component inspector 契约一致。
	traces := append([]navigationregistry.TraceRecord{}, inspection.Traces...)
	return apphttp.OK(c, NavigationInspectorSnapshot{
		Revision:          inspection.Snapshot.Revision,
		Digest:            inspection.Snapshot.Digest,
		SafeMode:          inspection.Snapshot.SafeMode,
		NavigationCount:   len(inspection.Snapshot.Navigation),
		RegionCount:       len(inspection.Snapshot.Regions),
		ProviderConflicts: len(inspection.Snapshot.NavigationConflicts) + len(inspection.Snapshot.RegionConflicts),
		Traces:            traces,
	})
}

func parseInspectorLimit(raw string, defaultLimit, maximum int) (int, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return defaultLimit, nil
	}
	limit, err := strconv.Atoi(raw)
	if err != nil || limit < 1 || limit > maximum {
		return 0, errInspectorLimit
	}
	return limit, nil
}
