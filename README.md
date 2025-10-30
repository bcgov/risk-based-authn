# Risk Assessment Engine 🛡️

This project is a robust risk assessment engine designed to evaluate and manage risks associated with various events. It provides a flexible and configurable framework for defining risk assessment rules, integrating with external services, and aggregating risk scores. The engine supports multiple risk assessment strategies and can be easily extended to incorporate new rules and services. It solves the problem of efficiently and accurately assessing risks in real-time, enabling proactive security measures and informed decision-making.

## 🚀 Key Features

- **Configurable Risk Rules**: Define risk assessment rules using a YAML configuration file (`rules.yaml`). Supports various rule types like velocity and denylist.
- **Real-time Risk Assessment**: Processes incoming events and evaluates risks based on the defined rules.
- **External Service Integration**: Integrates with external services like NATS (for message publishing) and Redis (for data storage and rate limiting).
- **Concurrent Processing**: Executes risk assessment handlers concurrently to minimize latency.
- **Flexible Risk Strategies**: Supports multiple risk aggregation strategies, such as `average` and `override`.
- **Graceful Shutdown**: Handles graceful shutdown of the server to prevent data loss.
- **Health Check Endpoint**: Provides a `/health` endpoint for monitoring the server's health.
- **CORS Support**: Handles Cross-Origin Resource Sharing (CORS) to allow requests from different domains.
- **Denylist Support**: Block specific IPs or IP ranges using the denylist rule.
- **Velocity Support**: Rate limit actions from specific IPs using the velocity rule.

## 🛠️ Tech Stack

- **Backend**:
    - Go
- **Configuration**:
    - YAML (`rules.yaml`)
- **HTTP Router**:
    - `github.com/go-chi/chi/v5`
- **Middleware**:
    - `github.com/go-chi/chi/v5/middleware`
- **CORS**:
    - `github.com/go-chi/cors`
- **Message Queue**:
    - NATS (`github.com/nats-io/nats.go`)
- **Data Store**:
    - Redis (`github.com/redis/go-redis/v9`)
- **Environment Variables**:
    - `github.com/joho/godotenv`
- **YAML Parsing**:
    - `gopkg.in/yaml.v3`

## 📦 Getting Started

### Prerequisites

- Go (version 1.20 or higher)
- Docker (for running Redis and NATS locally)

If you use [asdf](https://asdf-vm.com/) there is a tool-versions file with the correct golang version. To install:
- `asdf plugin add golang https://github.com/asdf-community/asdf-golang.git`
- `asdf install`

### Running Locally

1.  Create a `.env` file (optional) to configure environment variables. Example:

    ```
    PORT=8080
    ```

2. Run services:
`docker-compose up`

3. Live reload the application:
`make watch`


### Build and Run Application
- `make build`
- `make run`

### Run Tests
`make test`

## 💻 Usage

Send a POST request to the `/event` endpoint with a JSON payload containing the event data.

Example:

```json
{
  "event": "login",
  "data": {
    "ip": "192.168.1.1"
  }
}
```

The server will process the event, evaluate the risk, and return a response with the risk score.

## 📂 Project Structure

```
├── cmd
│   └── api
│       └── main.go         # Main application entry point
├── internal
│   └── server
│       ├── routes.go       # Defines HTTP routes and request handlers
│       └── server.go       # Defines the HTTP server and its configuration
├── rules
│   ├── denylist.go     # Implements the denylist risk rule
│   ├── import.go       # Loads and parses risk rule configurations
│   ├── velocity.go     # Implements the velocity risk rule
├── services
│   ├── natsClient.go   # Manages the NATS client connection
│   ├── redisClient.go  # Manages the Redis client connection
├── util
│   ├── constants.go    # Defines constant values
│   ├── helpers.go      # Provides utility functions
│   └── types.go        # Defines custom data types
├── rules.yaml          # Configuration file for risk assessment rules
├── go.mod              # Go module definition
├── go.sum              # Go module checksums
└── README.md           # This file
```


## 🤝 Contributing

Contributions are welcome! Please follow these steps:

1.  Fork the repository.
2.  Create a new branch for your feature or bug fix.
3.  Make your changes and commit them with descriptive messages.
4.  Push your changes to your fork.
5.  Submit a pull request.

## 💖 Thanks

Thank you for checking out this project! We hope it's helpful for your risk assessment needs.

This README is written by [readme.ai](https://readme-generator-phi.vercel.app/), the AI-powered README generator.
