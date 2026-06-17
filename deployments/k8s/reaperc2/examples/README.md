# Secret templates (safe to commit)

Files here use **placeholders only**. Copy to `*.local.yaml`, edit, apply the **`.local`** files only.

```bash
cp documentdb-secret.yaml documentdb-secret.local.yaml
cp documentdb-admin-secret.yaml documentdb-admin-secret.local.yaml
cp admin-bootstrap-secret.yaml admin-bootstrap-secret.local.yaml
# Optional — egress lockdown for ReaperC2 pods (requires a CNI that enforces NetworkPolicy):
# cp networkpolicy-egress-restricted.yaml networkpolicy-egress-restricted.local.yaml
# Edit all .local files, then apply (see ../README.md).
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

## `admin-bootstrap-secret.local.yaml`

| Key | Rule |
|-----|------|
| `username` | First admin UI login when no operators exist in MongoDB |
| `password` | Stored as Argon2id on first startup only |

Not used for DocumentDB. After the first operator exists, bootstrap env vars are ignored on later restarts (you can delete the secret).

## Bedrock (Operator AI)

See [bedrock-irsa.md](bedrock-irsa.md) and [bedrock-iam-policy.json](bedrock-iam-policy.json). Do not grant Bedrock on the EKS node group role unless you accept cluster-wide exposure.

For ECR pull credentials, use the commands in `registry-secret.yaml` (no Secret manifest in git).

## `networkpolicy-egress-restricted.local.yaml` (optional)

Template: [`networkpolicy-egress-restricted.yaml`](networkpolicy-egress-restricted.yaml). Copy to `networkpolicy-egress-restricted.local.yaml` and **replace the DocumentDB `ipBlock` CIDR** (placeholder `10.0.0.0/8`) with a range that actually reaches your DocumentDB endpoint. Add egress rules if you need extra ports or private endpoints.

Apply with **`../deploy.sh --with-egress all`** or **`../deploy.sh --with-egress apply-core`** after editing the `.local` file. Without a CNI that enforces policies, this manifest has no effect or may not behave as expected — see the **deploy.sh, reroll.sh, and egress** section in [../README.md](../README.md).

## Operator AI

Copy [`../operator-ai.yaml`](../operator-ai.yaml) → [`../operator-ai.local.yaml`](../operator-ai.local.yaml) (gitignored), edit ConfigMap + Secret, then:

```bash
kubectl apply -f ../operator-ai.local.yaml
kubectl rollout restart deployment/reaperc2-deployment -n reaperc2-ns
```
