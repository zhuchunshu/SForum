package hostapi

import (
	"strings"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
	"google.golang.org/grpc"
)

func TestGatewayPublishesDatabaseCatalogOnlyToFutureBrokers(t *testing.T) {
	first, err := NewPostgresProtocolV2DatabaseRuntime(new(pgxpool.Pool), []ProtocolV2DatabaseQueryDefinition{
		databasePublishQuery("demo.publish", "1.0.0", strings.Repeat("a", 64)),
	}, nil)
	if err != nil {
		t.Fatal(err)
	}
	gateway := NewGateway(nil)
	if err := gateway.BindProtocolV2DatabaseRuntime(first); err != nil {
		t.Fatal(err)
	}
	firstBroker := grpc.NewServer()
	gateway.RegisterProtocolV2(firstBroker)
	oldEngine := gateway.database

	second, err := NewPostgresProtocolV2DatabaseRuntime(new(pgxpool.Pool), []ProtocolV2DatabaseQueryDefinition{
		databasePublishQuery("demo.publish", "2.0.0", strings.Repeat("b", 64)),
	}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if err := gateway.PublishProtocolV2DatabaseRuntime(second); err != nil {
		t.Fatal(err)
	}
	if gateway.database == oldEngine {
		t.Fatal("future broker snapshot was not replaced")
	}
	if len(oldEngine.queries) != 1 || len(gateway.database.queries) != 1 {
		t.Fatalf("old/new query catalogs = %d/%d", len(oldEngine.queries), len(gateway.database.queries))
	}
	for key := range oldEngine.queries {
		if key.extensionVersion != "1.0.0" {
			t.Fatalf("active broker catalog changed: %#v", key)
		}
	}
	for key := range gateway.database.queries {
		if key.extensionVersion != "2.0.0" {
			t.Fatalf("future broker catalog = %#v", key)
		}
	}
	secondBroker := grpc.NewServer()
	gateway.RegisterProtocolV2(secondBroker)
}

func databasePublishQuery(extensionID, version, digest string) ProtocolV2DatabaseQueryDefinition {
	return ProtocolV2DatabaseQueryDefinition{
		ExtensionID: extensionID, ExtensionVersion: version, PackageDigest: digest,
		OperationID: extensionID + ".items.query", StatementVersion: "1",
		Scope: ProtocolV2DatabaseOwnSchema, SQL: "SELECT id FROM items",
		ResultSchemaID: extensionID + ".item", ResultSchemaVersion: "1",
		Columns: []ProtocolV2DatabaseColumn{{Name: "id"}}, MaxRows: 10,
	}
}
