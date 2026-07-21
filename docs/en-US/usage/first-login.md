# First registration & super admin

[← Usage](./README.md)

## Identity model

SForum uses **one user system**. Members, moderators, and admins differ by roles and permissions—not separate account tables.

| Role | Meaning |
| --- | --- |
| **First registered user** | Becomes protected `super_admin` |
| **`member`** | Default role after open registration; stable key, customizable display name |
| Other built-in roles | Moderator / operator templates in admin |

## Suggested first-run checklist

1. Open the site (dev default <http://127.0.0.1:3000>).  
2. Register the **first** account.  
3. Open admin: `/control-panel`.  
4. Confirm site name, locale, and appearance under site / personalization settings.  
5. Configure mail (built-in SMTP plugin) for password reset and notifications.  
6. Create categories and tags before inviting others (or adjust registration policy).

## Super-admin protection

The initial `super_admin` cannot be deleted, disabled, or stripped of super-admin powers. Grant other admins carefully.

## Sessions & security

- Browser sessions are server-backed (Redis).  
- Account security UI manages devices and revocation.  
- Production: rotate secrets and DB passwords in `.env.production`.

## Human verification (ALTCHA)

- Optional on registration and other scenarios.  
- Often disabled by default in development.  
- Toggle per scenario in admin security / verification settings.

## Next

- [Admin control panel](./admin.md)  
- [Forum day-to-day](./forum.md)  
