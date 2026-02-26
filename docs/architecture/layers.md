# Net layer → `server/`

### What it does:

Only accepts connections.
Nothing else.

Example in Aegis:

```go
listener, _ := net.Listen("unix", socket)
conn, _ := listener.Accept()
```

This layer:

* Doesn’t know what a health check is
* Doesn’t know what a session is
* Doesn’t know what a component is

It only accepts connections.
It’s like the physical door of the building.

---

# Transport layer → `router.go`

Here the incoming message is interpreted.

It receives something like:

```json
{
  "type": "HEALTH_CHECK",
  "payload": "data"
}
```

And decides:

→ “Ah, this goes to the health handler.”

This layer:

* Decodes JSON
* Decides which function to call
* Doesn’t do business logic

It’s like the receptionist saying:

> “Ah, you want to talk to the system health department.”

But doesn’t fix anything themselves.

---

# Adapter layer → `handlers/`

Now it gets interesting.

The handler:

* Receives the command
* Calls the correct service
* Returns a response over the network

Example:

```go
func HandleHealth(cmd Command, conn net.Conn) {
    result := healthService.Check(cmd.Payload)
    json.NewEncoder(conn).Encode(result)
}
```

The handler:

* Knows it’s a health check
* But doesn’t know how it works internally

It’s just a translator between:
Network ↔ Internal logic

---

# Business logic → `services/`

This is where the real logic lives.

Example:

```go
func Check(target string) HealthResponse {
    if target == "data" {
        return checkData()
    }
}
```

Here you:

* Validate filesystem
* Check internal state
* Query registries
* Perform calculations

This layer:

* Doesn’t care about network
* Doesn’t care about JSON
* Doesn’t care about sockets

It’s pure system logic.
This is the operational brain.

---

# Domain/Core → `core/`

This is the deepest layer.

Here live:

* Models
* Entities
* Base types
* Fundamental rules

Example in Aegis:

```go
type Component struct {
    Name string
    Requires []string
}
```

Or:

```go
func GetCPUInfo()
```

This layer:

* Doesn’t know there’s a daemon
* Doesn’t know there’s a CLI
* Doesn’t know there’s a router

It’s pure knowledge.
It’s the physics of the system.

---

# The correct mental model

Think like this:

```
Network enters
↓
Server accepts
↓
Router decides
↓
Handler translates
↓
Service executes logic
↓
Core contains knowledge
```

Each layer knows less about the outside world.
The deeper you go, the purer it is.

---

# Why this matters

If tomorrow you change:

* Unix socket → HTTP
* CLI → gRPC
* JSON → Protobuf

Does your internal logic break?

If yes → your architecture is wrong.
If no → the layers are correctly separated.

---

# In your specific case (Aegis)

Server:
Opens the Unix socket

Router:
Decides if it’s SESSION_START or HEALTH_CHECK

Handlers:
Receive the command and call services

Services:
SessionManager
ComponentRegistry
HealthService

Core:
System info
Component model
Base entities

---

# Simple rule to know where to put code

Ask yourself:

Does this code need to know a socket exists?
→ Goes in a handler.

Could this code be used if we remove the daemon tomorrow?
→ Goes in a service.

Does this code define what a component is or how data is validated?
→ Goes in core.
