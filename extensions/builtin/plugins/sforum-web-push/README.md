# sforum.web-push

Protected built-in **reference provider** for `notification.channel.web_push`.

## Ownership boundary

| Concern | Owner |
| --- | --- |
| Subscriptions, policy, durable delivery, the minimal service worker, per-user preferences | **Host** |
| VAPID credentials and RFC Web Push transport (`github.com/SherClockHolmes/webpush-go`) | **This plugin** |

## Features

- Browser Notification subscriptions require a matching live browser and Host
  subscription before they show as enabled;
- revoked device history is hidden from users;
- untouched Core reply/mention/moderation Web Push policy defaults to the
  enabled recommendation without overwriting operator choices;
- same-user tabs elect one SSE leader for realtime delivery, so the
  four-connection guard is not hit under ordinary multi-tab use.

## Configuration

Configure in **Admin → Notifications → Channels**: select the Web Push channel,
manage VAPID keys, and use the admin self-test to verify delivery. Operators can
per-channel reset to recommended defaults; test mail / test notifications are
excluded from cooldown and rate limits.
