# SForum Web Push

Protected built-in reference provider for `notification.channel.web_push`.
Core owns subscriptions, policy, durable delivery, and the minimal service
worker. This plugin owns VAPID credentials and RFC Web Push transport through
`github.com/SherClockHolmes/webpush-go`.
