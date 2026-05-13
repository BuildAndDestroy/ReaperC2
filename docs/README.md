# ReaperC2 documentation

ReaperC2 is a C2 framework with a **beacon HTTP API** and an **operator admin panel** backed by MongoDB. This wiki mirrors the [`docs/`](https://github.com/BuildAndDestroy/ReaperC2/tree/main/docs) folder in the main repository.

## Sections

| Topic | Description |
|-------|-------------|
| [Installation](/documentation/installation) | Prerequisites, building from source, MongoDB seeding |
| [Usage](/documentation/usage) | Environment variables, listeners, operator UI overview |
| [Docker Compose](/documentation/docker-compose) | Local full stack with MongoDB |
| [Kubernetes](/documentation/kubernetes) | Cluster deployment, manifests, admin access patterns |

Signed-in operators can also open **Documentation** in the left nav of the admin UI to read the same pages in the browser.

## Source of truth

- **Repository:** Markdown under `docs/` in the ReaperC2 repo.
- **Wiki:** Updated by the **Sync docs to Wiki** GitHub Action when `docs/*.md` changes on the default branch (requires a one-time wiki init and `WIKI_PUSH_TOKEN` secret).
