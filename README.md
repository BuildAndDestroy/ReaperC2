# ReaperC2
C2 framework that works on kubernetes and the cloud

<h1 align="center">
<br>
<img src=Screenshots/reaper-marauder.png >
<br>
ReaperC2
</h1>


## Example - Testing

### Docker Compose (full stack)

[`docker-compose.yml`](docker-compose.yml) runs **MongoDB 7** and a **ReaperC2** container (beacon **8080**, admin **8443**) on a shared network. Copy [`.env.example`](.env.example) to `.env`, set passwords, then:

**Scythe submodule (`third_party/Scythe`):** for a **pinned** Scythe revision matching this repo, run `git submodule update --init --recursive` before building (or use the helper below). The **`Dockerfile`** can **clone** Scythe from GitHub during `docker build` if `third_party/Scythe` is missing—so plain `docker compose up --build` works without a manual submodule step, but the clone uses **`SCYTHE_GIT_REF`** (default **`main`**) and may differ from the submodule commit until you init the submodule or set build args.

**Recommended (submodule + compose):**

```bash
./scripts/compose-up.sh
```

This runs `git submodule update --init --recursive`, then `docker compose up --build` (passes through extra args, e.g. `-d`).

**Or** without the script:

```bash
git submodule update --init --recursive   # optional if you want pinned Scythe in the image
docker compose up --build
```

Set **`SCYTHE_GIT_REF`** (e.g. in `.env` or the shell) to change the branch/tag used **only** when the image build clones Scythe. After updating the Scythe submodule pointer in git, **rebuild** container images and any **Scythe.embedded** beacons you deployed earlier.

- Admin UI: `http://127.0.0.1:8443/login` — first operator comes from `ADMIN_BOOTSTRAP_*` in `.env` when the `operators` collection is empty.
- MongoDB is also published on **27017** for local tools (override with `MONGO_HOST_PORT` in `.env`).
- The app connects with the Mongo **root** user and `MONGO_AUTH_SOURCE=admin` (see [`pkg/dbconnections/mongoconnections.go`](pkg/dbconnections/mongoconnections.go)); change `MONGO_USERNAME` / `MONGO_PASSWORD` / `MONGO_AUTH_SOURCE` if you switch to an application user. Passwords with `?`, `@`, and other URI characters are supported (credentials are URL-encoded when building the connection string).
- If Mongo auth fails after you change `MONGO_ROOT_PASSWORD` in `.env`, the `mongo_data` volume was probably initialized with an older password — run `docker compose down -v` once (wipes local DB data), then `docker compose up --build` again.
- **Scythe embedded binary:** the image is based on **`golang`** (includes `go` at runtime). `docker-compose.yml` sets `REAPERC2_ROOT=/root` so Scythe sources under `third_party/Scythe` resolve inside the container. After generating a beacon, use **Download Scythe.embedded** on the Beacons page to test the full flow.
- **Operator AI (Ollama on the host):** run Ollama on your machine (`ollama serve` or the desktop app), then in `.env` set `REAPER_AI_OLLAMA_ENABLED=1`, `REAPER_AI_OLLAMA_API_URL=http://host.docker.internal:11434/v1`, and list models you have pulled (`ollama list`, e.g. `REAPER_AI_OLLAMA_MODELS=gpt-oss:latest`). On **Mac/Windows Docker Desktop**, use `host.docker.internal` only — do **not** add `extra_hosts: host.docker.internal:host-gateway` (it breaks Ollama with `EOF`). On **Linux**, optional `COMPOSE_PROFILES=ollama-host` uses socat on the docker bridge; set `REAPER_AI_OLLAMA_API_URL=http://172.17.0.1:11434/v1` instead. See [Operator AI](docs/operator-guide-ai.md) and [Docker Compose](docs/docker-compose.md).

All helper scripts and the `mongoclient` image live under [`test/`](test/).

### One-shot local Mongo seed (recommended)

[`test/run_tests.sh`](test/run_tests.sh) creates a Docker network, starts **MongoDB Community** in a container, waits until it is ready, builds the **mongoclient** image, and runs [`test/setup_mongo.sh`](test/setup_mongo.sh) inside that image. It is **non-interactive** and suitable for CI.

```bash
cd test
./run_tests.sh
```

By default the Mongo container is removed when the script exits. To leave it running on `localhost:27017` for manual work:

```bash
KEEP_MONGO=1 ./run_tests.sh
```

Useful environment variables (both scripts honor the overlapping ones):

| Variable | Purpose |
|----------|---------|
| `MONGO_HOST` / `MONGO_PORT` | Mongo host and port (defaults: in-cluster service DNS for `setup_mongo.sh`; `run_tests.sh` sets host to the Mongo container name on the test network) |
| `MONGO_ADMIN_USER` / `MONGO_ADMIN_PASSWORD` | Root user for seeding (defaults match Docker `MONGO_INITDB_*` in `run_tests.sh`) |
| `MONGO_API_USER` / `MONGO_API_PASSWORD` | Application user created in `api_db` (defaults: `api_user` / `api_mongoApiPassword`) |
| `IMPORT_DATA_JSON` | Set to `0` to skip importing [`test/data.json`](test/data.json) |
| `DATA_JSON` | Path to JSON array file for `mongoimport` (default: `test/data.json` beside the script) |
| `DATA_JSON_COLLECTION` | Target collection for that import (default: `seed_docs`) |
| `DOCKER_NETWORK` / `MONGO_CONTAINER` | Override Docker network name and Mongo container name in `run_tests.sh` |
| `KEEP_MONGO` | `1` = do not remove the Mongo container on exit |
| `KEEP_TEST_NETWORK` | `1` = skip removing the test Docker network when cleaning up (only if `KEEP_MONGO` is not used) |

[`test/setup_mongo.sh`](test/setup_mongo.sh) creates `api_db` with `clients`, `heartbeat`, and `data` collections (plus indexes and sample documents). [`test/data.json`](test/data.json) is imported as **extra** seed documents into `seed_docs`; it does not replace the scripted fixture data.

**Kubernetes:** exec into a pod that has `mongosh` and this repo (or use the mongoclient image), then point at your cluster service, for example:

```bash
export MONGO_HOST=mongodb-service.reaperc2-ns.svc.cluster.local
export MONGO_PORT=27017
./setup_mongo.sh
```

**Manual Docker** (if you do not use `run_tests.sh`): build and run from `test/` with `MONGO_HOST` set to a resolvable hostname for the Mongo container on the same Docker network.

### Server

The server reads Mongo settings from environment variables (see [`pkg/dbconnections/mongoconnections.go`](pkg/dbconnections/mongoconnections.go)). After seeding with the defaults above, run locally against Docker Mongo on the published port:

```bash
export DEPLOY_ENV=ONPREM
export MONGO_HOST=127.0.0.1
export MONGO_PORT=27017
export MONGO_USERNAME=api_user
export MONGO_PASSWORD=api_mongoApiPassword
export MONGO_DATABASE=api_db
# Optional: when the DB user lives in the admin DB (e.g. root user)
# export MONGO_AUTH_SOURCE=admin

cd cmd && env GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -v -o ReaperC2
./ReaperC2
```

Example log lines:

```
Connected to MongoDB!
Beacon API listening on :8080
Admin panel listening on :8443
```

### Admin panel (same binary, second listener)

The process serves **two HTTP listeners**: the **beacon API** (implants / Scythe) and an **operator web UI** for signing in and creating `clients` rows (MongoDB) with a generated Scythe example.

| Variable | Purpose |
|----------|---------|
| `BEACON_ADDR` | Beacon API bind address (default `:8080`) |
| `ADMIN_ADDR` | Admin panel bind address (default `:8443`) |
| `ADMIN_BOOTSTRAP_USERNAME` / `ADMIN_BOOTSTRAP_PASSWORD` | If **no** operators exist in MongoDB, create the first account on startup (password stored as **Argon2id**). Omit to create operators manually in the `operators` collection. |
| `BEACON_PUBLIC_BASE_URL` | Default C2 base URL for Scythe examples when none is set per beacon (default `http://127.0.0.1:8080`, no path). Override per generation with **Beacon C2 base URL** on the Beacons page or `beacon_base_url` in `POST /api/beacons`. |
| `BEACON_PIVOT_PROXY` | Optional default `host:port` for Scythe `--proxy` when the beacon has a **parent** (pivot). Per-beacon override: **Pivot proxy** field or `pivot_proxy` in the generate API. |
| `SCYTHE_SRC` | Optional absolute path to [Scythe](https://github.com/BuildAndDestroy/Scythe) (`go.mod` + `./cmd`). If unset, ReaperC2 searches `REAPERC2_ROOT/third_party/Scythe`, then paths next to the **running binary** (covers `/root/cmd/ReaperC2` → `/root/third_party/Scythe` in Docker), then `Getwd()/third_party/Scythe`. |
| `REAPERC2_ROOT` | Optional; if set, Scythe is `$REAPERC2_ROOT/third_party/Scythe`. Sample Docker Compose / K8s YAML sets `/root` for the default image; **not strictly required** if the binary path alone resolves correctly. |
| `ADMIN_SESSION_TTL_HOURS` | Server-side session lifetime (default `168`). |
| `ADMIN_COOKIE_SECURE` | Set to `true` if the admin UI is only served over HTTPS (adds `Secure` on session cookies). |
| `ADMIN_DISABLE` | Set to `1` to run **only** the beacon listener (no admin port). |
| `ADMIN_ARGON2_TIME` | Argon2id time cost (default `3`). |
| `ADMIN_ARGON2_MEMORY_KIB` | Argon2id memory in KiB (default `65536`, i.e. 64 MiB). |
| `ADMIN_ARGON2_THREADS` | Argon2id parallelism (default `4`). |

Operator passwords are stored as **Argon2id** (serialized in `operators.password_hash`). **Existing bcrypt hashes** (`$2a$` / `$2b$`) still verify so you can migrate gradually.

Open `https://<host>:8443` (or `http://` locally; `/` redirects to **Engagements**). Pick a workspace, then use **Beacons**, **Commands**, and the other operator pages. Full per-page documentation is in [`docs/operator-guide.md`](docs/operator-guide.md) and in the admin UI under **Documentation → Operator guide**.

| Area | Purpose |
|------|---------|
| **Engagements** | Workspaces that scope beacons, commands, reports, topology, notes, and chat; assign operators (admins). |
| **Beacons** | Generate clients, Scythe Http options, **Scythe.embedded** download (`POST /api/beacons/scythe-embedded`; Go required on server), saved profiles, kill queue. |
| **Commands** | Queue tasks; stage uploads; view pending queue, artifacts, and output history. File bytes persist in MongoDB/DocumentDB GridFS (`reaper_artifacts`). |
| **Reports** | JSON / CSV / Ghostwriter / ATT&CK Navigator layer exports. |
| **Topology** | Interactive beacon graph (liveness + pivot chain). |
| **Notes & ATT&CK** | Engagement notes and MITRE Navigator layer source. |
| **Chat** | Operator chat per engagement (`operator_chat`). |
| **Engagement logs** | Audit trail for the active engagement. |
| **Users** (admins only) | Portal accounts (`/users`, `POST /api/users`). |
| **All logs** (admins only) | Global audit + JSON / Ghostwriter export (includes operator chat). |

**Roles** (field `operators.role` in MongoDB): **Admin** — full portal access including user management. **Operator** — beacons, reports, topology, chat, and profile management; **cannot** create users or call user APIs. Accounts without `role` are treated as **Admin** for backward compatibility. The bootstrap account is always **Admin**.

### Client

* Using a client, such as Scythe, we query the API

```
$ ./Scythe Http --method GET --timeout 5s --url http://127.0.0.1:8080 --headers 'Content-Type:application/json,X-Client-Id:550e8400-e29b-41d4-a716-446655440000,X-API-Secret:mysecurekey1' --directories '/heartbeat/550e8400-e29b-41d4-a716-446655440000,/heartbeat'
```

With a pivot (parent beacon), the example adds `--proxy <host:port>` (from the form, or `BEACON_PIVOT_PROXY`).

* If there is no authenticated user, then no access.

## Building the container image

### Makefile — AWS ECR (amd64 + arm64)

The root [`Makefile`](Makefile) builds a **multi-arch** image (`linux/amd64` and `linux/arm64`) with Docker **buildx** and pushes to **`registry.reaper-ut.com`** (override `ECR_REGISTRY` if needed).

**Prerequisites**

- [Docker](https://docs.docker.com/get-docker/) with the **buildx** plugin
- [AWS CLI](https://aws.amazon.com/cli/) v2 (`aws sts get-caller-identity` works after auth below) — used to mint the registry password
- [Git](https://git-scm.com/) (submodule init)

**Authenticate to the Reaper registry**

Two steps: set AWS CLI variables in the shell, then Docker-login to `registry.reaper-ut.com`. `make build` / `./ship.sh` also login, but you can (and should) do this yourself first.

**1. AWS CLI variables**

```bash
unset AWS_PROFILE

export AWS_REGION=us-east-1
export ECR_REPOSITORY=reaperc2
export ECR_REGISTRY=registry.reaper-ut.com

# Temporary creds from SSO / access portal / assume-role (keys start with ASIA):
export AWS_ACCESS_KEY_ID="ASIA..."
export AWS_SECRET_ACCESS_KEY="..."
export AWS_SESSION_TOKEN="..."
# IAM user keys (AKIA…): omit AWS_SESSION_TOKEN.

aws sts get-caller-identity
```

Named profile instead of key exports: `aws sso login --profile my-sso`, then `export AWS_PROFILE=my-sso` (profile **name**, not a 12-digit account id).

**2. Docker login**

```bash
aws ecr get-login-password --region "${AWS_REGION}" | \
  docker login --username AWS --password-stdin "${ECR_REGISTRY}"
```

Expect `Login Succeeded`. Then push (`make build-amd64`) and deploy (step 3).

The cluster pull secret is `reaperc2-myregistrykey` (`./deploy-cluster.sh ecr-secret`), with `--docker-server=registry.reaper-ut.com`.

**3. Deploy (cluster already running)**

Kubernetes does **not** watch `:latest`. After a successful push to `registry.reaper-ut.com/reaperc2:latest`, apply the overlay and restart pods so `imagePullPolicy: Always` pulls the new digest.

`kubectl` must be pointed at the EKS cluster (`aws eks update-kubeconfig` if needed). DocumentDB secrets and the first `./deploy.sh all` are already done for a live cluster — do not re-run `all` or `job-docdb-user`.

```bash
cd deployments/k8s/reaperc2

# Shortcut (apply overlay + rollout restart + wait):
./reroll.sh --apply-core
# same thing: ./ship.sh --deploy-only

# Or raw kubectl:
kubectl apply -k overlays/aws-ecr
kubectl rollout restart deployment/reaperc2-deployment -n reaperc2-ns
kubectl rollout status deployment/reaperc2-deployment -n reaperc2-ns --timeout=300s
kubectl get pods -n reaperc2-ns -o wide
```

Pods should become `Running` on a new age. Image on the pod should be `registry.reaper-ut.com/reaperc2:latest`. Reconnect admin with:

```bash
kubectl port-forward -n reaperc2-ns deployment/reaperc2-deployment 8443:8443
```

Build **and** deploy in one shot (rebuilds even if you just pushed): `./ship.sh`.

**Daily loop (always `:latest`)**

```bash
# after AWS vars + docker login above
make build-amd64                          # or: ./ship.sh --push-only
cd deployments/k8s/reaperc2 && ./reroll.sh --apply-core
```

On Apple Silicon, `go mod download` inside Docker buildx often crashes (Go SIGSEGV under QEMU). **`make build`** / **`./ship.sh`** cross-compile on your Mac (`make vendor` + `make build-binaries`), then Docker only packages the image (`Dockerfile.pack`). Use **`make build-docker`** on native Linux CI after **`make vendor`**.

**Pin a release tag** (still also aliases `:latest` and the git SHA):

```bash
make build IMAGE_TAG=v1.0.0
```

**Other registry hosts**

```bash
make build ECR_REGISTRY=registry.example.com AWS_REGION=us-east-1
```

**Makefile targets**

| Target | Description |
|--------|-------------|
| `make help` | List targets and current `IMAGE` |
| `make build` | Host cross-compile + multi-arch push to `ECR_REGISTRY` (recommended on Mac) |
| `make build-docker` | Full Docker build with vendored modules (Linux CI) |
| `make build-binaries` | Only `bin/linux-amd64` and `bin/linux-arm64/ReaperC2` |
| `make vendor` | `go mod vendor` (required before `build-docker`) |
| `make push` | Same as `make build` |
| `make build-amd64` | Push only `...:$(IMAGE_TAG)-amd64` (default tag `latest`) |
| `make build-arm64` | Push only `...:$(IMAGE_TAG)-arm64` |
| `make build-local` | Build `reaperc2:local` for your machine (`--load`, no registry) |
| `make ecr-login` | `get-login-password` → `docker login $(ECR_REGISTRY)` |
| `make ecr-create-repo` | Create the AWS ECR repository if `ECR_REGISTRY` is an `*.dkr.ecr.*` host |
| [`./ship.sh`](deployments/k8s/reaperc2/ship.sh) | Push `:latest` (amd64) + apply-core + rollout |

**Variables** (override on the command line or in the environment)

| Variable | Default | Purpose |
|----------|---------|---------|
| `ECR_REGISTRY` | `registry.reaper-ut.com` | Docker login and push host |
| `ECR_REPOSITORY` | `reaperc2` | Repository name (`registry.reaper-ut.com/reaperc2:latest`) |
| `IMAGE_TAG` | `latest` | Image tag; every push also aliases the git SHA, and aliases `:latest` if you override this |
| `AWS_REGION` | `us-east-1` | Region for `aws ecr get-login-password` |
| `SCYTHE_GIT_REF` | `main` | Branch/tag when the Dockerfile must clone Scythe (submodule preferred) |
| `AWS_ACCESS_KEY_ID` | (unset) | Access key (`ASIA…` STS or `AKIA…` IAM user) |
| `AWS_SECRET_ACCESS_KEY` | (unset) | Secret key paired with `AWS_ACCESS_KEY_ID` |
| `AWS_SESSION_TOKEN` | (unset) | **Required** for temporary/`ASIA` creds; omit for IAM user keys |
| `AWS_CLI_PROFILE` | (unset) | Named `aws --profile` for registry login; takes precedence over env keys |
| `AWS_PROFILE` | (unset) | Shell profile if no `AWS_CLI_PROFILE` and no env keys. Must be a profile **name**, never an account id. Ignored when `AWS_ACCESS_KEY_ID` is set. |

**Deploy to EKS after push**

First install: follow [`deployments/k8s/reaperc2/README.md`](deployments/k8s/reaperc2/README.md#quick-install-script) (`./deploy.sh all`, DocumentDB Jobs, `apply-ingress`). After that, **do not edit the image tag** — keep `…/reaperc2:latest` and run [`./ship.sh`](deployments/k8s/reaperc2/ship.sh).

### Docker build (single arch, any registry)

For local tags or a registry other than ECR:

```bash
git submodule update --init --recursive   # recommended: embed exact Scythe commit from this repo
docker build -t reaperc2:latest .
```

The Dockerfile uses `TARGETARCH` (default `amd64`). For arm64 on a single platform: `docker build --build-arg TARGETARCH=arm64 -t reaperc2:arm64 .`

If `third_party/Scythe` is missing, the image build **clones** Scythe using `SCYTHE_GIT_REF` (default `main`):

```bash
docker build --build-arg SCYTHE_GIT_REF=your-tag -t reaperc2:latest .
```

Push to your registry, then point **`deployments/k8s/**`** manifests at that tag.

### `DEPLOY_ENV` (runtime, not compile time)

`DEPLOY_ENV` tells ReaperC2 **where it is running** so it can adjust behavior—today mainly the MongoDB/DocumentDB connection string ([`pkg/dbconnections/mongoconnections.go`](pkg/dbconnections/mongoconnections.go)):

| Value | Effect |
|-------|--------|
| `AWS` | Adds DocumentDB TLS URI params (`tls`, `replicaSet=rs0`, CA file path, etc.) |
| `ONPREM` | Standard Mongo URI; optional `MONGO_USE_TLS=true` |
| `AZURE` / `GCP` | Placeholders in [`cmd/main.go`](cmd/main.go) (“coming soon”) |

Valid values: `AWS`, `AZURE`, `GCP`, `ONPREM` ([`pkg/deploymehere/deploymehere.go`](pkg/deploymehere/deploymehere.go)). Invalid or unset → treated as `ONPREM`.

The Dockerfile default is `ONPREM` for local Compose. **You do not need to bake `AWS` into the ECR image** for EKS: [`deployments/k8s/reaperc2/base/deployment.yaml`](deployments/k8s/reaperc2/base/deployment.yaml) already sets `DEPLOY_ENV=AWS`, which overrides the image default at pod start.

To change the image default at build time (optional):

```bash
docker build --build-arg DEPLOY_ENV=AWS -t reaperc2:aws .
# or
docker buildx build --build-arg DEPLOY_ENV=AWS ...
```

Prefer **runtime** config (Kubernetes `env`, Compose `environment`) so one image tag works everywhere and you can change behavior without rebuilding.

## Example - Kubernetes

### Kubernetes: public beacon, admin on localhost (tunnel)

ReaperC2 listens on **two ports** in one process by default: the **beacon API** on **8080** and the **operator admin UI** on **8443** (see `BEACON_ADDR` / `ADMIN_ADDR` in the table below). For cluster deployments you should treat them differently:

- **Beacons / Scythe:** expose **8080** to the Internet via your Ingress or load balancer (TLS termination in front of the Service is fine). The sample manifests under [`deployments/k8s/`](deployments/k8s/) wire **only** port **8080** on `reaperc2-service` to the public host.
- **Admin panel:** do **not** put **8443** on a public Ingress or load balancer. Instead, from a trusted workstation, open a **tunnel** to **8443** on the ReaperC2 Pod or Deployment and use the UI at **`http://127.0.0.1:8443`** on that machine.

**Typical: `kubectl port-forward`**

```bash
kubectl port-forward -n reaperc2-ns deployment/reaperc2-deployment 8443:8443
```

Leave that process running, then in a browser on the same machine open **`http://127.0.0.1:8443/login`**. The admin server uses plain HTTP on that port unless you change the binary or put TLS in front of it yourself.

To forward to a specific Pod (useful if the Deployment name differs):

```bash
kubectl port-forward -n reaperc2-ns pod/$(kubectl get pod -n reaperc2-ns -l app=reaperc2-deployment -o jsonpath='{.items[0].metadata.name}') 8443:8443
```

Adjust **`-n`**, **labels**, and **Deployment/Pod** names to match your YAML.

**Jump host: SSH local forward after `port-forward` on the bastion**

If your laptop cannot reach the Kubernetes API directly but you can SSH to a bastion that has `kubectl` and kubeconfig:

1. On the bastion, run **`kubectl port-forward … 8443:8443`** as above (it listens on the bastion’s loopback).
2. From your laptop: **`ssh -N -L 8443:127.0.0.1:8443 user@bastion.example.com`**
3. Open **`http://127.0.0.1:8443/login`** on the laptop.

**Beacon base URL for implants**

Configure **`BEACON_PUBLIC_BASE_URL`** (and/or each beacon’s **Beacon C2 base URL** in the UI) to the **public** origin that beacons should call—e.g. `https://c2.example.com` where your Ingress terminates and forwards to **8080**. That must **not** be `http://127.0.0.1:8443`; localhost is only for operators via the tunnel.

**Optional:** set **`ADMIN_DISABLE=1`** on the workload if you want **no** admin listener at all (beacon-only); you lose the web UI unless you run a separate pattern.

### Requirements

* Kubernetes cluster (e.g. EKS) with `kubectl` configured
* Container image in a registry (for AWS: `make build` → ECR; see [Building the container image](#building-the-container-image))
* Traefik (or another ingress) for **beacon** traffic on **8080**
* A public hostname for implants (`BEACON_PUBLIC_BASE_URL`); keep admin **8443** off public ingress—use [port-forward](#kubernetes-public-beacon-admin-on-localhost-tunnel) instead

### Manifest layout

| Path | Use when |
|------|----------|
| [`deployments/k8s/`](deployments/k8s/) | EKS or **k3s** + **DocumentDB** + Traefik/cert-manager ([`ship.sh`](deployments/k8s/reaperc2/ship.sh), [`deploy-cluster.sh`](deployments/k8s/reaperc2/deploy-cluster.sh), [`deploy.sh`](deployments/k8s/reaperc2/deploy.sh), [`reroll.sh`](deployments/k8s/reaperc2/reroll.sh); quick index [`deployments/k8s/DEPLOY.md`](deployments/k8s/DEPLOY.md); `kubectl apply -k deployments/k8s/AWS` still works as **aws-ecr** shim) |
| [`deployments/k8s/OnPrem/`](deployments/k8s/OnPrem/) | In-cluster MongoDB |
| [`deployments/k8s/full-deployment.yaml`](deployments/k8s/full-deployment.yaml) | Sample all-in-one with in-cluster Mongo |

**AWS deploy (summary)**

```bash
cd deployments/k8s/reaperc2
# First time: ./deploy.sh all  then job-docdb-user / job-docdb-init / apply-ingress
# Every image update:
./ship.sh
```

* Point **Ingress / IngressRoute** only at Service port **8080** (beacon).
* Set your subdomain and TLS issuer in `ingress.yaml` / `ingressroute.yaml` (staging vs prod cert-manager issuer).
* DocumentDB: split secret keys (`host`, `username`, …), not a single URI; run **`fetch-docdb-ca-bundle.sh`** then **`docdb-init-job.yaml`** for collections/indexes.
* **Operator AI on EKS:** copy `deployments/k8s/operator-ai.yaml` → `operator-ai.local.yaml`, apply locally (ConfigMap + secrets) — see AWS README and [Operator AI](docs/operator-guide-ai.md).

Full checklist (read [DocumentDB pitfalls](deployments/k8s/reaperc2/README.md#documentdb-pitfalls-read-first) first): [`deployments/k8s/reaperc2/README.md`](deployments/k8s/reaperc2/README.md#run-from-scratch-checklist) and [`docs/kubernetes.md`](docs/kubernetes.md).