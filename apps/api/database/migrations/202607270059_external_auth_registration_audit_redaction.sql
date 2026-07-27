-- +goose Up
-- External registration audit history needs provider/operator correlation, not
-- an immutable package digest or any identity/OAuth material. Restrict this
-- repair to the affected action; extension artifact and other audit history
-- remain untouched.
UPDATE audit_events
SET metadata = metadata
  - 'ownerPackageDigest'
  - 'subjectDigest'
  - 'providerSubject'
  - 'rawSubject'
  - 'state'
  - 'codeVerifier'
  - 'completionToken'
  - 'code'
  - 'token'
  - 'secret'
  - 'clientSecret'
WHERE action = 'auth.external_register.success'
  AND metadata ?| ARRAY[
    'ownerPackageDigest', 'subjectDigest', 'providerSubject', 'rawSubject',
    'state', 'codeVerifier', 'completionToken', 'code', 'token', 'secret',
    'clientSecret'
  ];

-- +goose Down
-- Redacted sensitive material must never be reconstructed.
