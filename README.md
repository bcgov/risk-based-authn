# Project rba

This is an event-driven risk-based authentication engine, designed for flexible use with different authentication systems. For different authentication events, e.g. `login`, `refresh_token`, `logout`, rules can be flexibly configured to provide a risk score. Authentication systems can send data to this system on each event to retrieve a score. See [The configuration file](./rules.yaml) for an example.

## Getting Started

Some rules require additional services, e.g. redis. To run these locally:
`docker-compose up`

Local development with hot reloading:
Live reload the application:
```bash
make watch
```

Build the application
```bash
make build
```

Run the application
```bash
make run
```

Run the test suite:
```bash
make test
```

## Organization

- Each configurable rule has a corresponding entry in the [rules directory](./rules), including a parser and a handler. On a given event, each handler correspinding to it will run and a final score calculated.

- Supporting service connections (e.g. redis, nats) are in the [services](./services/) directory.

- Server logic is in the [server](./internal/server/) directory.

- The application launches from [main.go](./cmd/api/main.go)

