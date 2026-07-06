# 2026-07-07 Admin Database Table Manager

## Status

Accepted.

## Context

Operators need a lightweight phpMyAdmin-like way to inspect the SForum
PostgreSQL database from the existing admin panel. The feature is useful for
development and troubleshooting, but arbitrary SQL execution or data mutation
would add a large security and maintenance surface.

## Decision

Add a core admin database table manager as a read-only PostgreSQL browser.

- The feature is core admin tooling, not a plugin extension point.
- Access requires the dedicated `database.manage` permission.
- The first release lists non-system schemas only and excludes PostgreSQL
  system schemas such as `pg_catalog`, `information_schema`, and `pg_toast`.
- Operators can inspect tables, columns, indexes, constraints, paged rows,
  simple single-column filters, sorting, and masked CSV exports.
- Sensitive columns are detected by column-name patterns such as password,
  secret, token, credential, session, cookie, hash, salt, private key, access
  key, and refresh. Row lists and CSV exports mask those values by default.
- A masked sensitive cell can be revealed one at a time through a row-key based
  API when the table has a primary key.
- Do not add row mutation, table mutation, multi-database connection management,
  or a SQL console in v1.

## Consequences

- The API must treat catalog metadata as the source of truth for allowed table
  and column identifiers before building SQL.
- CSV export is a troubleshooting aid, not a backup path; it keeps sensitive
  values masked and caps exports.
- Future write features require a separate decision, stronger confirmation UI,
  audit events, and additional permission modeling.
