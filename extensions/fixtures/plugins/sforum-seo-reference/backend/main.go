package main

import (
	"context"
	"errors"
	"log"
	"strings"

	pluginv2 "github.com/zhuchunshu/sforum/apps/api/sdk/plugin/v2"
)

const titleContributionID = "sforum.seo-reference.title"

func main() {
	registry, err := pluginv2.NewSEORegistry(pluginv2.SEODefinition{
		ID: titleContributionID, ContractVersion: titleContributionID + "@1",
		Scope: "core.page.topic", Kind: "title", Action: "filter",
		Handler: titleContributionID, FailurePolicy: "fallback", TimeoutMS: 500,
		Execute: applyReferenceTitle,
	})
	if err != nil {
		log.Fatalf("configure SEO reference plugin: %v", err)
	}
	pluginv2.Serve(pluginv2.NewServer().WithSEORegistry(registry))
}

func applyReferenceTitle(ctx context.Context, call *pluginv2.SEOCall) (pluginv2.SEODocument, error) {
	switch call.Current.Title {
	case "reference:fail":
		return pluginv2.SEODocument{}, errors.New("reference SEO failure")
	case "reference:timeout":
		<-ctx.Done()
		return pluginv2.SEODocument{}, context.Cause(ctx)
	}
	result := call.Current
	result.Title = strings.TrimSpace(result.Title) + " | SEO Reference"
	return result, nil
}
