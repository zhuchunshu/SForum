package bootstrap

import (
	"encoding/json"
	"testing"
)

func TestProductionLifecycleStackPublishesCoreReservedOpenAPIAggregate(t *testing.T) {
	stack, _, _ := newBootstrapLifecycleStack(t)
	publication := stack.RouteSchemas.PublicationSnapshot()
	contract := stack.RouteSchemas.ContractSnapshot()
	if publication.Revision != 0 || publication.AggregateRevision == "" ||
		contract.Revision != publication.Revision || contract.AggregateRevision != publication.AggregateRevision ||
		len(contract.Document) == 0 || len(contract.Artifacts) != 0 || len(contract.GeneratedClientOperations) != 0 {
		t.Fatalf("production OpenAPI publication = %#v contract = %#v", publication, contract)
	}
	var document map[string]any
	if err := json.Unmarshal(contract.Document, &document); err != nil {
		t.Fatal(err)
	}
	if document["openapi"] != "3.1.0" || len(document["paths"].(map[string]any)) != 0 {
		t.Fatalf("production core aggregate = %#v", document)
	}
}
