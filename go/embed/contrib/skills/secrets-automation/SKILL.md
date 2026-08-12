---
name: secrets-automation
description: "Use when setting up unattended secret access (service accounts, vault scoping, non-echoing tokens)."
version: 1.0.0
license: MIT
metadata:
  package: unleash-skills
  author: NetVar1337/unleash
  category: imported
  upstream: C:\Users\Admin\.agents\skills\secrets-automation\SKILL.md
---

> Bundled with Unleash skills pack. Source: C:\Users\Admin\.agents\skills\secrets-automation\SKILL.md

# Secrets Automation

Use dedicated machine identities for unattended access. Never turn a human account password into an automation credential.

## Security invariants

- Never save, echo, log, memorize, script, or place a human vault password in command arguments, environment variables, hooks, shell history, or source control.
- Treat any password or token pasted into chat as exposed. Recommend rotation and do not repeat it.
- Prefer a dedicated service account restricted to a dedicated vault and only the required operations.
- Capture generated tokens without printing them. Prefer direct process capture, a concealed prompt, or a local clipboard-to-file transfer followed immediately by clipboard clearing.
- Do not claim success until a real secret-manager API operation works under the machine identity.
- If validation fails, remove the credential from the agent's active environment so it does not override a working interactive integration.

## Standard workflow

1. **Discover the supported automation identity.** Prefer service accounts, Connect-style servers, workload identity, or the vendor's supported SDK/MCP. Do not automate a master password.
2. **Create a dedicated vault or namespace.** Use a descriptive name such as `Hermes`; do not grant Personal/Private vault access by default.
3. **Grant least privilege.** Start with read-only. Add write access only when the agent must create or rotate items.
4. **Generate the token once.** Capture stdout or clipboard without displaying the value in tool output. Clear temporary files and clipboard content immediately.
5. **Store through the host's secret channel.** For Hermes skill credentials, use the Hermes secret environment path rather than project files. Preserve unrelated entries with an atomic targeted update.
6. **Verify the identity, state, and intended operation.** Use the vendor-supported identity command, then perform a harmless read. If write access is required, create and remove a non-secret verification item.
7. **Apply to long-running processes.** Restart only the relevant gateway/agent process after successful verification.
8. **Document scope, not secrets.** Report the identity name, vault scope, permissions, and verification status—never token contents.

## 1Password-specific rules

- `OP_SERVICE_ACCOUNT_TOKEN` is the supported non-interactive CLI credential.
- Verify service-account authentication with `op user get --me` and confirm `Type: SERVICE_ACCOUNT` and `State: ACTIVE`.
- `op service-account ratelimit` is a useful second server-side verification.
- Do not use `op vault list` as the primary service-account authentication check; it is not in the documented supported-command set.
- Desktop-app integration is intentionally interactive. On Windows, authorization is process/session scoped, expires after inactivity, has a hard lifetime, and uses Windows Hello. It cannot provide a supported guarantee of never prompting.
- If Connect variables are present, `OP_CONNECT_HOST` and `OP_CONNECT_TOKEN` take precedence over `OP_SERVICE_ACCOUNT_TOKEN`; inspect variable names without printing values.
- A locally decodable `op whoami` result is not sufficient proof that server-side service-account authentication works. Require `op user get --me` or another supported server request.

## Safe token handling pattern

- Keep the token out of command-line arguments.
- When a web UI offers only a Copy button, click it locally, read the clipboard inside a short-lived process, validate only the prefix/shape, atomically write the secret destination, and clear the clipboard.
- Never send a screenshot containing a generated token to vision or OCR output. If local OCR is needed to locate controls, redact long token-like strings before returning text and use accessibility labels such as “Copy … token” when available.
- Delete screenshots or temporary artifacts that could contain token pixels after setup.

## Failure handling

- Separate **identity creation** from **authentication verification**. An admin console showing Active and expected permissions does not prove that the captured token authenticates.
- Retry briefly for eventual consistency, then stop. Do not rotate repeatedly or create multiple service accounts without evidence.
- Test precedence conflicts and interactive-app interference without persisting extra credentials.
- Preserve the last securely backed-up token, but remove a non-working token from the active agent environment to avoid breaking other authentication paths.
- Escalate to the vendor/admin console with sanitized facts: CLI version, identity state, permission scope, supported verification command, and exit code—never the token.

## References

- See `references/1password-service-accounts.md` for authoritative command support, Windows prompt behavior, and a concise verification checklist.

