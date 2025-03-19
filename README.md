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

## Examples


### Database

* Spin up the test database to use collections

```
$ docker run --rm -it --name mongodb -p 27017:27017 -d mongodb/mongodb-community-server:latest
```

* Run the test script to setup mongodb collections

```
$ docker build -t mongoclient -f mongoclient.dockerfile .

$ docker run --rm -it  mongoclient
```

* Spin up the test client to work with the database

```
$ docker run --rm -it --entrypoint=bash  mongoclient

$ mongosh mongodb://172.17.0.2:27017
```

### Server 

* Build and run the Server

```
$ env GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -v -o ReaperC2

$ ./ReaperC2 
2025/03/17 22:41:52 Connected to MongoDB!
2025/03/17 22:41:52 Server running on port 8080...
```

### Client

* Using a client, such as Scythe, we query the API

```
$ ./Scythe Http --method GET --timeout 5s --url http://127.0.0.1:8080 --headers 'Content-Type:application/json,X-Client-Id: 550e8400-e29b-41d4-a716-446655440000,X-API-Secret: mysecurekey1' --directories '/heartbeat'
```

* If there is no authenticated user, then no access.
