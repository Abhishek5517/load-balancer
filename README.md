# L7 Application Layer Load Balancer (Go + Fiber)

A **Layer 7 (Application Layer) Load Balancer** built using **Golang (Fiber)** with a **custom, configurable Rate Limiter** implemented using **Redis + Lua scripts** based on the **Token Bucket algorithm**.


---

## 🚀 Features

- 🌐 **Layer 7 Load Balancing**
  - HTTP-aware request routing
  - Path & method based forwarding
  - Reverse proxy implementation

- ⚡ **High-Performance Go Fiber Server**
  - Built on top of `fasthttp`
  - Optimized for low latency and high throughput

- 🛑 **Custom Rate Limiter**
  - Implemented using **Redis + Lua**
  - **Token Bucket Algorithm**
  - Atomic operations using Lua scripts
  - Prevents race conditions across distributed instances

- 🔧 **Fully Configurable**
  - Tokens per second
  - Bucket capacity
  - Per-IP / Per-Client / Per-API limits
  - Easy to extend for user-based or header-based limits


---

