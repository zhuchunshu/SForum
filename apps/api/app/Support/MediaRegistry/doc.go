// Package mediaregistry defines SForum's immutable, exact-artifact Media
// Pipeline Registry contract.
//
// This package validates and plans MIME policy, security processors, metadata,
// transforms, variants, CDN selection, retention, and deletion hooks. It does
// not read media bytes, write storage, persist attachment state, enqueue River
// jobs, or coordinate extension lifecycle. Those production integrations must
// retain Host attachment authorization and use the admission, operation, and
// plan fences exposed here. Ordered operations require an injected Host-owned
// ReceiptAuthority plus opaque durable evidence; the Registry neither owns a
// signing secret nor treats its public SHA checksum as authority. Production
// must make source receipts transactional with immutable storage admission,
// back operation claims with shared durable CAS, and propagate quarantine
// across nodes; this in-process kernel cannot prove those integrations. A
// committed source/admission receipt gates the first background scanner, exact step
// receipts gate later work, and after_delete additionally requires an exact
// deletion-complete receipt. Plans are Host-private and refuse raw JSON
// serialization; inspectors must use the redacted PlanSummary projection.
package mediaregistry
