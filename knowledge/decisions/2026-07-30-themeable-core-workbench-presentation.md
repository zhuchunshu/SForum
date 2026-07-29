# Themeable Core Workbench Presentation

Date: 2026-07-30

## Context

Page Registry previously used `replaceable` for both plugin behavior
replacement and theme presentation. The moderation workbench correctly set it
to false, but that also forced the route onto Host fallback chrome and made the
active public theme irrelevant.

## Decision

Page definitions now expose two capabilities:

- `replaceable`: a trusted extension may replace the Core page after the
  existing exact-artifact approval flow.
- `themeable`: the selected theme may provide an L1 presentation template that
  embeds the page's required reviewed Host island.

All existing replaceable pages remain themeable for compatibility.
`moderation.review` is `replaceable=false, themeable=true` and requires
`sf-moderation-review`. Theme registration uses a theme-specific validation
path; plugin and lifecycle publication continue to require `replaceable`.

## Consequences

- Re-activating a theme can now update moderation chrome without moving review
  permissions, data, mutations, or rendering into the theme.
- Generic provider approval still rejects moderation replacement.
- Built-in themes must declare, digest, validate, and test the moderation
  template like every other themeable page.
