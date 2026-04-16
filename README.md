# L7 Application Layer Load Balancer (Go + Fiber)

A **Layer 7 (Application Layer) Load Balancer** built using **Golang (Fiber)** with four load balancing strategies, dynamic server registration, active/passive health checking, and a **custom Rate Limiter** implemented using **Redis + Lua scripts** based on the **Token Bucket algorithm**.

---

## Features

- **Four load balancing strategies** — Round Robin, Least Connection, Consistent Hashing, Random
- **Dynamic server pool** — register/deregister backends at runtime via HTTP API
- **Dual health checking** — active polling every 10s + passive detection on proxy failure
- **High-Performance Go Fiber Server** — built on top of `fasthttp`, optimized for low latency and high throughput
- **Custom Rate Limiter**
  - Implemented using **Redis + Lua**
  - **Token Bucket Algorithm**
  - Atomic operations using Lua scripts — prevents race conditions across distributed instances
  - Per-IP / Per-Client / Per-API limits, easy to extend

---

## Architecture

```
Client
  │
  ▼
Fiber HTTP Server (:4000)
  │
  ├─ POST /register        → add a backend server
  ├─ POST /deregister      → remove a backend server
  ├─ GET  /servers         → list servers + health status
  └─ ALL  *                → ClientRequest handler
                                │
                                ├─ Rate limiter (Redis token bucket)
                                │
                                ├─ Strategy selector
                                │    ├─ round-robin       (default)
                                │    ├─ least-connection
                                │    ├─ consistent-hashing (by client IP)
                                │    └─ random-server
                                │
                                └─ Reverse proxy (goroutine + channel)
                                        │
                                        └─ Backend servers
```

### Package layout

| File | Responsibility |
|---|---|
| `main.go` | Fiber app, route registration, request handler, strategy dispatch |
| `strategies/serverConfig.go` | `Server` struct — URL, health flag, active connection counter |
| `strategies/serverPool.go` | `Pool` — thread-safe server list, round-robin counter, hash ring integration |
| `strategies/roundRobinStrategy.go` | `RoundRobinServer()` — wraps `Pool.Next()` |
| `strategies/leastConnection.go` | `LeastConnectionServer()` — picks server with fewest active connections |
| `strategies/consistentHashing.go` | `HashRing` with 150 virtual nodes per server; `ConsistentHashServer(ip)` |
| `strategies/randomServer.go` | `RandomServer()` — picks a random healthy server |
| `strategies/healthCheck.go` | `StartHealthChecker()` — background goroutine polling `/health` on each server |
| `RedisClient/RedisInit.go` | Initialises `*redis.Client` from `REDIS_URL` env var |

### Request flow

1. Client sends any HTTP request to `:4000`
2. Rate limiter checks the client IP against a Redis token bucket — returns `429` if exceeded
3. Strategy selector (`selectServer`) picks a healthy backend
4. For `least-connection`, the active connection counter on that server is incremented
5. The request is forwarded in a goroutine; response is returned via channel
6. On proxy failure the server is immediately marked unhealthy (passive health check)
7. On success, `least-connection` counter is decremented and the response is returned to the client

### Health checking

- **Active** — `StartHealthChecker` polls `GET {server}/health` every 10 seconds and updates the pool
- **Passive** — any proxy error marks the server as unhealthy immediately

---

## Routing Strategies

### Round Robin (default)
Cycles through healthy servers in order using an atomic counter.

### Least Connection
Routes each request to the healthy server with the lowest number of in-flight connections, tracked with `atomic.Int32`.

### Consistent Hashing
Maps the client IP onto a hash ring with 150 virtual nodes per server (MD5-based). The same client IP always lands on the same backend (when healthy), which is useful for session affinity.

### Random
Picks a healthy server at random on each request using `math/rand`. No state required — simple and effective for even distribution at scale.

---

## Getting Started

### Prerequisites

- Go 1.21+
- Redis running on `localhost:6379`

### Configuration

Copy `.env.example` to `.env` (or create `.env`) and set:

```env
PORT=:4000
REDIS_URL=redis://localhost:6379/0
STRATEGY=round-robin   # round-robin | least-connection | consistent-hashing | random-server
```

### Run

```bash
go mod tidy
go run main.go
```

### Build

```bash
go build -o load-balancer
./load-balancer
```

---

## API

### Register a backend server

```http
POST /register
Content-Type: application/json

{"url": "http://localhost:9000"}
```

### Deregister a backend server

```http
POST /deregister
Content-Type: application/json

{"url": "http://localhost:9000"}
```

### List servers

```http
GET /servers
```

Returns a JSON array with each server's URL, health status, and active connection count.

---

## Dependencies

| Package | Purpose |
|---|---|
| `github.com/gofiber/fiber/v2` | HTTP framework |
| `github.com/joho/godotenv` | `.env` loading |
| `github.com/Abhishek5517/Rate-Limiter` | Token bucket rate limiter (Redis + Lua) |
| `github.com/redis/go-redis` (via RedisClient) | Redis client for rate limiter state |
