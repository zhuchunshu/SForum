# Notifications

[<- Usage overview](./README.md)

Notification Platform V2 keeps the inbox and delivery policy in Core while
letting trusted extensions declare notification types or provide external
channels.

## Member settings

Open `/settings/notifications` to review each available notification type and
channel. A control can be:

- **Follow site recommendation**: inherit the operator's current default.
- **Enabled** or **disabled**: keep an explicit personal override when the
  operator allows it.
- **Locked/unavailable**: the type is required, disabled by the operator, or
  its selected provider is unavailable.

Restoring recommendations deletes personal overrides; it does not change site
policy or provider secrets. Inbox filtering and unread counts remain available
when an external channel is unavailable.

## Browser push

The Web Push section reports browser support and permission state. Enabling it
registers SForum's Host-owned worker at `/_sforum/notifications/sw.js` with the
scope `/_sforum/notifications/`. The worker only displays a bounded notification
and opens the same-origin notifications page. It never imports plugin code or
accepts arbitrary actions or external URLs.

The API associates a subscription only with the signed-in user. The settings
page lists active devices only; revoked rows remain available to server-side
audit and delivery relationships. Settings and delivery views redact the
endpoint and browser key material. Core reply, mention, and moderation Web Push
defaults to the enabled recommendation without overwriting operator-edited
policy. Revoking browser permission or a listed subscription does not disable
the in-app inbox.

## Operator settings

Operators with `settings.notifications.manage` use
`/control-panel/settings/notifications` to:

- enable types and channels, set recommendations, and allow user overrides;
- review Core and extension ownership, including newly declared plugin types
  that default to disabled;
- select the exact Web Push provider artifact, run a self-test, and inspect
  redacted delivery health;
- restore Host recommendations without clearing VAPID or other provider
  secrets.

Mail transport and mail history remain under Mail. Existing mail APIs and
`mail_deliveries` are preserved; Web Push uses the generic channel-delivery
ledger. A provider failure is visible there and does not suppress in-app
delivery.

## Troubleshooting

- If Web Push is unavailable, confirm browser support, notification permission,
  an active subscription, a selected provider, and configured VAPID keys.
- If a policy save conflicts, reload the page before retrying; settings use an
  optimistic revision.
- A hidden or deleted topic/comment intentionally appears as unavailable and
  exposes no actor, excerpt, review note, or route.
