# ReaperC2
C2 framework that works on kubernetes and the cloud

<h1 align="center">
<br>
<img src=Screenshots/reaper-marauder.png >
<br>
ReaperC2
</h1>


## Work In Progress

This C2 is currently in development.
Would do not recommend using this server until a stable release is built.
Currently only uses commands, we will need to integrate better calls

## Example - Testing

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

cd cmd && env GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -v -o ReaperC2
./ReaperC2
```

Example log lines:

```
Connected to MongoDB!
Server running on port 8080...
```

### Client

* Using a client, such as Scythe, we query the API

```
$ ./Scythe Http --method GET --timeout 5s --url http://127.0.0.1:8080 --headers 'Content-Type:application/json,X-Client-Id: 550e8400-e29b-41d4-a716-446655440000,X-API-Secret: mysecurekey1' --directories '/heartbeat'
```

* If there is no authenticated user, then no access.

## Example - Kubernetes

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