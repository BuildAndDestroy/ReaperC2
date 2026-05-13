# Installation

## Prerequisites

- **Go** `1.23.x` matching [`go.mod`](https://github.com/BuildAndDestroy/ReaperC2/blob/main/go.mod) if you build locally.
- **MongoDB** reachable from the ReaperC2 process (single replica is typical for small deployments).
- Optional: **git** with submodule support if you want the **pinned** [Scythe](https://github.com/BuildAndDestroy/Scythe) tree under `third_party/Scythe` (recommended for reproducible **Scythe.embedded** builds).

## Clone the repository

```bash
git clone https://github.com/BuildAndDestroy/ReaperC2.git
cd ReaperC2
git submodule update --init --recursive   # recommended for Scythe
```

## Build the binary

```bash
cd cmd && CGO_ENABLED=0 go build -o ReaperC2
```

The same binary serves **two HTTP listeners**: beacon API (default `:8080`) and admin panel (default `:8443`).

## MongoDB setup

ReaperC2 expects application collections in a database such as `api_db`. For local Docker-based seeding, use the scripts under [`test/`](https://github.com/BuildAndDestroy/ReaperC2/tree/main/test):

```bash
cd test
./run_tests.sh
```

See [`test/setup_mongo.sh`](https://github.com/BuildAndDestroy/ReaperC2/blob/main/test/setup_mongo.sh) for created users, indexes, and sample data. Point `MONGO_HOST` / `MONGO_PORT` at your instance when not using the default test network.

## First operator

- **Bootstrap (recommended for first boot):** set `ADMIN_BOOTSTRAP_USERNAME` and `ADMIN_BOOTSTRAP_PASSWORD` when the `operators` collection is empty; the server creates an **Admin** account on startup (password stored as Argon2id).
- **Manual:** insert an operator document into MongoDB with a compatible password hash (see README in the repo for Argon2id / bcrypt notes).

## Environment essentials

At minimum set `MONGO_HOST`, `MONGO_PORT`, `MONGO_USERNAME`, `MONGO_PASSWORD`, `MONGO_DATABASE`, and `DEPLOY_ENV` (e.g. `ONPREM`). Optional `MONGO_AUTH_SOURCE` when the user authenticates against the `admin` database (common for root users).

Full variable tables: [Usage](/documentation/usage) and the root [README](https://github.com/BuildAndDestroy/ReaperC2/blob/main/README.md).

## Next steps

- [Docker Compose](/documentation/docker-compose) for a batteries-included local stack.
- [Kubernetes](/documentation/kubernetes) for cluster deployment.
- [Usage](/documentation/usage) for day-to-day operation.
