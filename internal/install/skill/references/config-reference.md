# frizzle.json Configuration Reference

## Top-level schema

```json
{
  "port": 4000,
  "buses": [...]
}
```

| Field | Type | Required | Default | Description |
|---|---|---|---|---|
| `port` | int | No | 4000 | HTTP port to listen on |
| `buses` | Bus[] | No | `[{"name":"default","rules":[]}]` | Event buses with rules and targets |

## Bus

```json
{
  "name": "default",
  "rules": [...]
}
```

| Field | Type | Required | Description |
|---|---|---|---|
| `name` | string | Yes | Bus identifier — matched against `EventBusName` in SDK entries |
| `rules` | Rule[] | Yes | Rules evaluated top-to-bottom |

## Rule

```json
{
  "name": "order-events",
  "description": "Forward order events to the order service",
  "pattern": {
    "source": ["myapp.orders"],
    "detail-type": ["OrderPlaced", "OrderCancelled"]
  },
  "targets": [...]
}
```

| Field | Type | Required | Description |
|---|---|---|---|
| `name` | string | Yes | Human-readable rule identifier |
| `description` | string | No | What this rule does |
| `pattern` | object | Yes | EventBridge pattern to match events against |
| `targets` | Target[] | Yes | Where to forward matching events |

**Rule evaluation:** Rules are checked top-to-bottom. When an event matches a rule's pattern, all of that rule's targets fire (fan-out). Multiple rules can match the same event.

**Empty pattern:** `"pattern": {}` matches every event. Use this for catch-all rules at the bottom.

## Target

### Log target

```json
{ "type": "log" }
```

Prints the full event to frizzle's stdout.

### HTTP target

```json
{
  "type": "http",
  "url": "http://localhost:3001/webhook",
  "method": "POST",
  "headers": {
    "X-Source": "frizzle",
    "Authorization": "Bearer test-token"
  },
  "timeout": "5s"
}
```

| Field | Type | Required | Default | Description |
|---|---|---|---|---|
| `type` | string | Yes | Must be `"http"` |
| `url` | string | Yes | Downstream endpoint URL |
| `method` | string | No | `"POST"` | HTTP method |
| `headers` | object | No | `{}` | Extra headers to include |
| `timeout` | string | No | `"10s"` | Request timeout (Go duration format: `"5s"`, `"500ms"`, `"1m"`) |

The event is sent as the request body with `Content-Type: application/json`. The body is the EventBridge event envelope (snake_case fields, detail as an object).

## Full example — multi-bus config

```json
{
  "port": 4000,
  "buses": [
    {
      "name": "default",
      "rules": [
        {
          "name": "order-events",
          "description": "Forward order events to the order service",
          "pattern": {
            "source": ["myapp.orders", "myapp.checkout"],
            "detail-type": ["OrderPlaced", "OrderCancelled"]
          },
          "targets": [
            { "type": "log" },
            {
              "type": "http",
              "url": "http://localhost:3001/webhook",
              "method": "POST",
              "timeout": "5s"
            }
          ]
        },
        {
          "name": "high-value-orders",
          "description": "Alert on orders above $500",
          "pattern": {
            "detail-type": ["OrderPlaced"],
            "detail": {
              "amount": [{"numeric": [">=", 500]}]
            }
          },
          "targets": [
            { "type": "log" },
            {
              "type": "http",
              "url": "http://localhost:3002/alerts",
              "method": "POST"
            }
          ]
        },
        {
          "name": "catch-all",
          "description": "Log any events that don't match other rules",
          "pattern": {},
          "targets": [{ "type": "log" }]
        }
      ]
    },
    {
      "name": "analytics",
      "rules": [
        {
          "name": "user-events",
          "pattern": {
            "source": [{"prefix": "myapp.users"}]
          },
          "targets": [
            { "type": "log" },
            {
              "type": "http",
              "url": "http://localhost:3003/analytics",
              "method": "POST"
            }
          ]
        }
      ]
    }
  ]
}
```
