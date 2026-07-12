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
}
