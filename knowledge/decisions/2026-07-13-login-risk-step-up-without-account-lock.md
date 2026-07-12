# Login Risk Uses Step-up Verification, Not an Account Lock

Date: 2026-07-13
Status: Accepted

## Context

The original login limiter hard-locked the whole account after
`maxFailures * 3` failures. An attacker could distribute attempts across IPs
and deny the victim access even though no single source crossed its limit.

## Decision

- Keep temporary hard locks for the account+IP pair and the IP dimension.
- At the higher account threshold, write a short-lived `verification_required`
  marker instead of an account lock.
- Check this marker only after a valid password and active-user check. Wrong
  passwords keep the generic credential error and cannot enumerate risk state.
- A valid `login_risk` human-verification result retries the same login, clears
  the account marker/current pair failures, and establishes the session.
- Redis keys use hashes of normalized login and IP inputs. Redis errors retain
  the existing fail-open behavior so limiter availability cannot block login.

## Consequences

Distributed attacks cannot lock every source for a victim. Pair and IP limits
still slow brute-force and password spraying, while users who know the correct
password have an explicit recovery path. Deployments that enable ALTCHA for
`login_risk` get the step-up challenge through the existing provider contract.
