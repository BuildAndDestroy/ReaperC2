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
- The app connects with the Mongo **root** user and `MONGO_AUTH_SOURCE=admin` (see [`pkg/dbconnections/mongoconnections.go`](pkg/dbconnections/mongoconnections.go)); change `MONGO_USERNAME` / `MONGO_PASSWORD` / `MONGO_AUTH_SOURCE` if you switch to an application user.
- **Scythe embedded binary:** the image is based on **`golang`** (includes `go` at runtime). `docker-compose.yml` sets `REAPERC2_ROOT=/root` so Scythe sources under `third_party/Scythe` resolve inside the container. After generating a beacon, use **Download Scythe.embedded** on the Beacons page to test the full flow.

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
| `REAPER_ARTIFACT_DIR` | Directory for staged operator uploads and files pulled from beacons via Scythe’s `download` built-in (default `./data/reaper_artifacts`). Metadata is in MongoDB collection `file_artifacts`. |
| `ADMIN_SESSION_TTL_HOURS` | Server-side session lifetime (default `168`). |
| `ADMIN_COOKIE_SECURE` | Set to `true` if the admin UI is only served over HTTPS (adds `Secure` on session cookies). |
| `ADMIN_DISABLE` | Set to `1` to run **only** the beacon listener (no admin port). |
| `ADMIN_ARGON2_TIME` | Argon2id time cost (default `3`). |
| `ADMIN_ARGON2_MEMORY_KIB` | Argon2id memory in KiB (default `65536`, i.e. 64 MiB). |
| `ADMIN_ARGON2_THREADS` | Argon2id parallelism (default `4`). |

Operator passwords are stored as **Argon2id** (serialized in `operators.password_hash`). **Existing bcrypt hashes** (`$2a$` / `$2b$`) still verify so you can migrate gradually.

Open `https://<host>:8443/beacons` (or `http://` locally; `/` redirects to **Beacons**). The UI includes:

| Area | Purpose |
|------|---------|
| **Beacons** | Generate clients (optional **Beacon C2 base URL**: `http`/`https`, FQDN or IP, optional port — saved on the profile for embedded rebuilds). Optional label, `ParentClientId` for pivot chain, optional pivot proxy for Scythe. **Scythe Http** options in the UI match `Http` subcommand flags (`-method`, `-timeout`, `-body`, `-directories`, `-headers`, `-proxy`, `-skip-tls-verify`); **HTTP client timeout** is separate from **phone-home interval** (seconds). **Download Scythe.embedded** runs `go build` on the vendored Scythe submodule (`third_party/Scythe`; clone with `git submodule update --init`) and streams the binary — the admin host needs **Go** installed. API: `POST /api/beacons/scythe-embedded`. Each generation **always saves a profile** in `beacon_profiles`. List/delete saved profiles. |
| **Commands** | Queue beacon tasks: strings (e.g. `whoami`, `download <host path>`) or JSON objects for Scythe file ops (`command_obj`, or stage a file with `POST /api/beacon-staging` then queue with `upload.staging_id` + `remote_path`). Heartbeat returns a JSON `Commands` array (strings and/or objects). Pulled files are stored under `REAPER_ARTIFACT_DIR` and listed/downloaded from this page (`GET /api/beacon-artifacts`, `GET /api/beacon-artifacts/{id}/file`). |
| **Reports** | Download JSON or CSV exports (redacted or full). JSON includes `command_output` (recent beacon command results from the `data` collection). **Ghostwriter CSV** (`/api/reports/export-ghostwriter`) uses the same 13-column schema as Logs for clients, saved profiles, and command output—no operator chat (chat is under Logs). |
| **Topology** | Graph of C2 → beacons (and parent → child when `ParentClientId` is set on a client). |
| **Chat** | Operator messages stored in `operator_chat`. |
| **Users** (admins only) | Create additional portal accounts and assign **Admin** or **Operator** (`/users`, `POST /api/users`). |
| **Logs** (admins only) | View recent **audit** events (`audit_logs`): operator actions plus **beacon** deliveries (`beacon_commands_delivered`) and **output** (`beacon_output_received`). Download JSON (`/api/logs/export`, includes `operator_chat`) or **Ghostwriter CSV** (`/api/logs/export-ghostwriter`: audit + beacon results + operator chat) for Specter Ops Ghostwriter. |

**Roles** (field `operators.role` in MongoDB): **Admin** — full portal access including user management. **Operator** — beacons, reports, topology, chat, and profile management; **cannot** create users or call user APIs. Accounts without `role` are treated as **Admin** for backward compatibility. The bootstrap account is always **Admin**.

### Client

* Using a client, such as Scythe, we query the API

```
$ ./Scythe Http --method GET --timeout 5s --url http://127.0.0.1:8080 --headers 'Content-Type:application/json,X-Client-Id:550e8400-e29b-41d4-a716-446655440000,X-API-Secret:mysecurekey1' --directories '/heartbeat/550e8400-e29b-41d4-a716-446655440000,/heartbeat'
```

With a pivot (parent beacon), the example adds `--proxy <host:port>` (from the form, or `BEACON_PIVOT_PROXY`).

* If there is no authenticated user, then no access.

## Example - Kubernetes

Build the same image locally or in CI (network required if the build must **clone** Scythe because the submodule was not checked out):

```bash
git submodule update --init --recursive   # recommended: embed exact Scythe commit from this repo
docker build -t reaperc2:latest .
```

Or rely on the Dockerfile clone: `docker build -t reaperc2:latest .` (uses `SCYTHE_GIT_REF`, default `main`). Push to your registry, then point **`deployments/k8s/**`** manifests at that tag. CI should either run **`git submodule update --init --recursive`** before **`docker build`**, or set **`--build-arg SCYTHE_GIT_REF=…`** to match the Scythe revision you intend to ship.

### Requirements

* Kubernetes Cluster
* Traefik routing - Update routing from deployments/k8s/full-deployment.yaml if you are using something else
* A domain for your http(s) requests

### Yaml Updates

* Add your subdomain to the full-deployment.yaml
* Add your docker registry secret to full-deployment.yaml
* Add your secrets that match your golang binary to allow the connections to mongodb to work
* Apply the yaml:

```
$ kubectl apply -f full-deployment.yaml 
namespace/reaperc2-ns created
secret/reaperc2-myregistrykey created
secret/reaperc2-mongodb-secrets created
service/mongodb-service created
persistentvolume/mongo-pv created
persistentvolumeclaim/mongo-pvc created
deployment.apps/mongodb-deployment created
deployment.apps/reaperc2-deployment created
service/reaperc2-service created
ingress.networking.k8s.io/reaperc2-ingress created
ingressroute.traefik.io/reaperc2-ingressroute created
```

* Assuming all works, delete the deployment
* On line 191, change the following in your full-deployment.yaml for a signed cert

```
    cert-manager.io/cluster-issuer: letsencrypt-prod
    # cert-manager.io/cluster-issuer: letsencrypt-staging
```

* Note: We leave staging set to true to avoid timing out your domain due to accidents

* Your C2 is now running