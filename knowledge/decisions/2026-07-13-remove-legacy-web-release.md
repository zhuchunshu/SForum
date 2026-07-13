# Remove Legacy Web Release and Runtime Frontend Builds

Date: 2026-07-13
Status: Implemented

## Context

SForum has not shipped. The repository briefly carried two extension frontend
models: a host-built trusted Vue registry/Web Release pipeline and the newer
buildless Settings Document plus prebuilt-component model. Keeping both added
worker queues, database tables, release storage, Bun/Web dependencies in API
images, runtime supervisors, polling UI, permissions, and two trust lifecycles.
There is no production installation that requires that compatibility surface.

## Decision

Delete the old execution path completely:

- no `frontend.admin` or `frontend.layer` Manifest fields;
- no runtime compilation of Vue SFCs, extension dependency install, Nuxt Layer,
  static admin registry, Web Release coordinator, builder, storage, worker, or
  admin release screen;
- no release-specific permission, option, queue, environment variable, volume,
  OpenAPI route, SDK slot API, fixture, or deployment dependency;
- plugin enable/disable and theme activation return the resulting `Extension`
  synchronously;
- public themes use only `theme.json`, package assets, and Page Registry;
- settings use Schema UI/Actions, or a package-local prebuilt `.mjs`/`.css`
  component after exact digest trust and explicit administrator confirmation.

The forward cleanup migration remains only to remove tables/options/permissions
from development databases that already ran the discarded migrations. Fresh
installs never create those tables.

## Trust Boundary

Component code is fully trusted after approval. The durable grant binds
extension id, version, Admin Micro-frontend API version, component id, and
`adminFrontendDigest`. Assets are served only from the installed package by an
authenticated immutable digest endpoint. Changed bytes invalidate old trust;
missing/revoked/invalidated trust or load failure falls back to Schema UI.

Confirmation codes prove intentional authorization only. Backend permissions,
namespaced API routes, settings/action allowlists, secret encryption/masking,
audit, and lifecycle checks remain authoritative.

## Consequences

- Operators never build SForum when installing, configuring, enabling, or
  switching an extension.
- API and worker images no longer contain Bun, node_modules, or Web source.
- There is one admin component model and one public theme activation model.
- A future host release system, if needed, must be designed as host deployment
  infrastructure and cannot reuse extension manifests as arbitrary build input.
