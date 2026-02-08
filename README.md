# What is this?

Hey y'all, i've been trying to get better w/ go and this is that!

I went w/ a very standard ECS pattern and I've been experimenting w/ using AI agents to implement some features w/ ~some, lol, success

Deploys happen on push via Cloud Build and I have a very basic client talking via websockets here

[https://dusty.wtf/projects/dmud/](https://dusty.wtf/projects/dmud/)

# ...

```
find internal -type f -name '*.go' -exec sh -c 'echo "=== {} ==="; cat {}' \;
```

| Target | What it does |
|---|---|
| `make` | Build the binary to `bin/dmud` |
| `make run` | Build and run locally |
| `make watch` | Hot-reload dev server (requires `air`) |
| `make test` | Run tests |
| `make test-race` | Run tests with race detector |
| `make vet` | Run `go vet` |
| `make race` | Run with race detector |
| `make clean` | Remove build artifacts |
| `make docker-build` | Build Docker image (`dmud:latest`) |
| `make docker-run` | Build and run in Docker (port 8080) |
| `make docker-stop` | Stop running dmud containers |
| `make docker-clean` | Remove the dmud image |







