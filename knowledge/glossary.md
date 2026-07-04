# Glossary

Add shared project terms here as they become stable.

Initial candidates:

- Forum - the overall discussion product.
- Category - a top-level grouping for topics.
- Topic - a discussion thread started by a user.
- Post - an individual message inside a topic.
- Moderator - a user with permission to manage content.
- User - one account in the forum identity system, regardless of whether the
  person is a regular member, moderator, or administrator.
- Role / User group - a named collection of permissions assigned to users.
- Permission - a stable action key checked by the API, such as
  `topic.create` or `role.manage`.
- Super administrator - the highest-privilege role. The first registered user
  becomes the protected initial super administrator.
- Member - the default system role for open registration. Its display alias can
  change, but the `member` role key is stable and the role is not deletable
  while it remains the default registration role.
