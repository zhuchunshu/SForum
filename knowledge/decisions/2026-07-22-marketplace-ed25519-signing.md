# Marketplace index signing: Ed25519

**Date:** 2026-07-22  
**Status:** superseded by `2026-10-24-app-store-out-of-tree.md`  
**Area:** V3 P12 Marketplace

> **Superseded 2026-10-24.** The Marketplace reserve package this record designed
> was deleted: it never gained an HTTP/CLI consumer, the app store moved
> out-of-tree, and `MARKETPLACE_ED25519_PUBLIC_KEY_HEX` no longer exists. Kept for
> historical reasoning only — Ed25519 index signing is not part of core.

## Context

The first Marketplace Support package used HMAC-SHA256 over the index body.
HMAC requires sharing a symmetric secret with every Host that verifies the
signed catalog. That is a poor fit for a multi-operator open-source framework:

- Secret distribution and rotation are operationally hard.
- A leaked Host secret can forge indexes for that deployment.
- Offline/air-gapped operators still need a public verification path.

## Decision

Marketplace indexes are signed with **standard Ed25519**:

- Private key stays with the index publisher (SForum release engineering or an
  operator-run private catalog).
- Public key + `signerId` are configured on the Host (`Ed25519Verifier`).
- Canonical body is JSON with the `signature` field cleared; signature is
  hex-encoded 64-byte Ed25519 output.
- `signerKind` defaults to `ed25519`. Other algorithms are rejected.

HMAC is **not** retained as a production path. Dev/test may set
`OperatorPolicy.AllowUnsigned` only outside production/staging.

## Consequences

- Operators distribute **public keys**, not shared secrets.
- Multiple publisher keys can be rotated by changing Host verifier config.
- Index tampering (including nested `dependencies` / `notices`) fails verify.
- Key compromise requires publisher re-sign + Host key update; Hosts do not
  hold signing material.
