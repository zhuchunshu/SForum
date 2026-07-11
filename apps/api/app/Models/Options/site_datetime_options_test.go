package options

import (
	"context"
	"testing"
	"time"
)

func TestSiteDateTimeOptionsPublicDefaults(t *testing.T) {
	service := NewServiceWithCacheTTL(&fakeStore{}, time.Minute)

	items, err := service.List(context.Background())
	if err != nil {
		t.Fatalf("List returned error: %v", err)
	}

	if got := adminValueFromPublic(items, NameSiteTimezone); got != recommendedSiteTimezone {
		t.Fatalf("expected default timezone %q, got %q", recommendedSiteTimezone, got)
	}
	if got := adminValueFromPublic(items, NameSiteDateFormat); got != recommendedSiteDateFormat {
		t.Fatalf("expected default date format %q, got %q", recommendedSiteDateFormat, got)
	}
	if got := adminValueFromPublic(items, NameSiteTimeFormat); got != recommendedSiteTimeFormat {
		t.Fatalf("expected default time format %q, got %q", recommendedSiteTimeFormat, got)
	}
	if got := adminValueFromPublic(items, NameSiteStartOfWeek); got != "1" {
		t.Fatalf("expected default start of week 1, got %q", got)
	}
}

func TestSiteDateTimeOptionsAcceptValidValues(t *testing.T) {
	store := &fakeStore{items: map[string]string{}}
	service := NewServiceWithCacheTTL(store, time.Minute)
	actor := settingsActor()

	_, err := service.UpdateMany(context.Background(), actor, []UpdateInput{
		{Name: NameSiteTimezone, Value: "Asia/Shanghai"},
		{Name: NameSiteDateFormat, Value: "Y/m/d"},
		{Name: NameSiteTimeFormat, Value: "g:i A"},
		{Name: NameSiteStartOfWeek, Value: "0"},
	})
	if err != nil {
		t.Fatalf("UpdateMany returned error: %v", err)
	}

	if store.items[NameSiteTimezone] != "Asia/Shanghai" {
		t.Fatalf("timezone not saved: %#v", store.items)
	}
	if store.items[NameSiteDateFormat] != "Y/m/d" {
		t.Fatalf("date format not saved: %#v", store.items)
	}
	if store.items[NameSiteTimeFormat] != "g:i A" {
		t.Fatalf("time format not saved: %#v", store.items)
	}
	if store.items[NameSiteStartOfWeek] != "0" {
		t.Fatalf("start of week not saved: %#v", store.items)
	}
}

func TestSiteDateTimeOptionsRejectInvalidValues(t *testing.T) {
	store := &fakeStore{items: map[string]string{}}
	service := NewServiceWithCacheTTL(store, time.Minute)
	actor := settingsActor()

	cases := []UpdateInput{
		{Name: NameSiteTimezone, Value: "Not/AZone"},
		{Name: NameSiteDateFormat, Value: "yyyy-MM-dd"},
		{Name: NameSiteTimeFormat, Value: "24h"},
		{Name: NameSiteStartOfWeek, Value: "7"},
	}
	for _, input := range cases {
		if _, err := service.UpdateMany(context.Background(), actor, []UpdateInput{input}); err == nil {
			t.Fatalf("expected rejection for %s=%q", input.Name, input.Value)
		}
	}
}

func TestNormalizeSiteTimezone(t *testing.T) {
	if _, ok := normalizeSiteTimezone("Asia/Shanghai"); !ok {
		t.Fatal("Asia/Shanghai should be valid")
	}
	if _, ok := normalizeSiteTimezone("UTC"); !ok {
		t.Fatal("UTC should be valid")
	}
	if _, ok := normalizeSiteTimezone(""); ok {
		t.Fatal("empty timezone should be invalid")
	}
	if _, ok := normalizeSiteTimezone("GMT+8"); ok {
		// Go LoadLocation 不接受固定偏移缩写作为 IANA 名。
		t.Fatal("GMT+8 should be invalid IANA name")
	}
}
