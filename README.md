# storage

How data gets stored and moved around the network

## Local environment

Run a local api and nostr-relay using Docker Deskop.

- nostr-relay: `ws://localhost:9000`
- api: `http://localhost:9001`

**Prerequisites**:

1. Install [Docker Desktop](https://docs.docker.com/compose/install/)

Start the containers:

```
make run
```

[Install](https://github.com/fiatjaf/noscl#installation) and configure `noscl` to interact with the relay

```
# Add the local relay (only required once)
noscl relay add ws://localhost:9000

# Generate and set a private key
noscl key-gen
noscl setprivate <your key>

# Send a test note
noscl publish 'hello, nostr!'
```
