# Secret templates (safe to commit)

Files here use **placeholders only**. Copy to `*.local.yaml`, edit, apply the **`.local`** files only.

```bash
cp documentdb-secret.yaml documentdb-secret.local.yaml
cp documentdb-admin-secret.yaml documentdb-admin-secret.local.yaml
# Edit both .local files, then apply (see ../README.md).
```

`*.local.yaml` is gitignored — never commit real hostnames or passwords.

## `documentdb-secret.local.yaml` rules

| Key | Rule |
|-----|------|
| `database` | App database name (e.g. `reaperc2_db`) |
| `auth_source` | **Must match `database`** unless your platform team says otherwise |
| `username` / `password` | App user; must match what DocumentDB has after `docdb-init-user-job` |

**Common mistake:** `database: reaperc2_db` with `auth_source: api_db` → authentication fails.

After you change the password in `.local.yaml`, re-run **`docdb-init-user-job`** (it updates the password if the user already exists). See [../README.md#sync-documentdb-password](../README.md#sync-documentdb-password).

For ECR pull credentials, use the commands in `registry-secret.yaml` (no Secret manifest in git).
