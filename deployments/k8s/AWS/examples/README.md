# Secret templates (safe to commit)

Files here use **placeholders only**. Before applying to a cluster:

```bash
cp documentdb-secret.yaml documentdb-secret.local.yaml
cp documentdb-admin-secret.yaml documentdb-admin-secret.local.yaml
# Edit *.local.yaml with your DocumentDB endpoint and passwords.
kubectl apply -f documentdb-secret.local.yaml
kubectl apply -f documentdb-admin-secret.local.yaml   # optional, for docdb-init-user-job
```

`*.local.yaml` is gitignored. Never commit real credentials.

For ECR pull credentials, use the commands in `registry-secret.yaml` (no Secret manifest in git).
