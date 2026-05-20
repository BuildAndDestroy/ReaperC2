# Account

**Path:** `/account`

Available to any signed-in operator. Does not require an active engagement.

- **Password** — Change password (minimum length enforced server-side).
- **Two-factor authentication (TOTP)** — Optional Google Authenticator–compatible 2FA: set up (QR + verify), or disable (requires current password).

If 2FA is enabled, login continues at `/login/mfa` after username/password.
