# frizzle

<p align="center"><img src="frizzle.png" alt="frizzle" width="200"/></p>

Local AWS Event Bus simulator for testing event-driven systems without an AWS account.

## Install

```
go install github.com/LukeOfEarth/frizzle@latest
```

## Quick start

Start frizzle (catches all events, logs them, and forwards to a downstream endpoint):

```
frizzle start --port 4000 --forward http://localhost:3001/webhook
```

### Point your AWS SDK at frizzle

Configure your EventBridge client to use frizzle's local endpoint instead of AWS:

**Go SDK v2**

```go
cfg, _ := config.LoadDefaultConfig(ctx,
    config.WithRegion("us-east-1"),
    config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider("test", "test", "")),
)
client := eventbridge.NewFromConfig(cfg, func(o *eventbridge.Options) {
    o.BaseEndpoint = aws.String("http://localhost:4000")
})
```

**JavaScript SDK v3**

```js
import { EventBridgeClient, PutEventsCommand } from "@aws-sdk/client-eventbridge";

const client = new EventBridgeClient({
  region: "us-east-1",
  endpoint: "http://localhost:4000",
  credentials: { accessKeyId: "test", secretAccessKey: "test" },
});
```

**Python boto3**

```python
import boto3

client = boto3.client("events",
    region_name="us-east-1",
    endpoint_url="http://localhost:4000",
    aws_access_key_id="test",
    aws_secret_access_key="test",
)
```

**Java SDK v2**

```java
EventBridgeClient client = EventBridgeClient.builder()
    .region(Region.US_EAST_1)
    .endpointOverride(URI.create("http://localhost:4000"))
    .credentialsProvider(StaticCredentialsProvider.create(
        AwsBasicCredentials.create("test", "test")))
    .build();
```

Now your application's `PutEvents` calls go to frizzle instead of AWS.

### Quick test with curl

```
curl -X POST http://localhost:4000/events/default \
  -H "Content-Type: application/json" \
  -d '{
    "Entries": [{
      "version": "0",
      "id": "evt-001",
      "detail-type": "OrderPlaced",
      "source": "myapp.orders",
      "detail": { "orderId": "ORD-123", "amount": 150 }
    }]
  }'
```

## Config file mode

For multiple buses, rules, and pattern matching:

```
frizzle init                # scaffold full example config
frizzle init --simple       # minimal single-bus config
frizzle start               # reads .frizzle/frizzle.json from cwd
```

### .frizzle/frizzle.json structure

```json
{
  "port": 4000,
  "buses": [
    {
      "name": "default",
      "rules": [
        {
          "name": "order-events",
          "pattern": {
            "source": ["myapp.orders"],
            "detail-type": ["OrderPlaced", "OrderCancelled"]
          },
          "targets": [
            { "type": "log" },
            {
              "type": "http",
              "url": "http://localhost:3001/webhook",
              "method": "POST",
              "headers": { "X-Source": "frizzle" },
              "timeout": "5s"
            }
          ]
        }
      ]
    }
  ]
}
```

Rules are evaluated top-to-bottom. Every matching rule fires all its targets (fan-out). A catch-all rule with an empty `pattern: {}` matches everything — place it last to log any unmatched events.

## Pattern matching

Supports the full EventBridge pattern syntax:

| Operator | Example |
|---|---|
| Exact string | `"source": ["myapp.orders"]` |
| Prefix | `{"prefix": "myapp.orders"}` |
| Suffix | `{"suffix": ".pdf"}` |
| Wildcard | `{"wildcard": "report_*.pdf"}` |
| Case-insensitive | `{"equals-ignore-case": "orderplaced"}` |
| Anything-but | `{"anything-but": ["val1", "val2"]}` or `{"anything-but": {"prefix": "prod."}}` |
| Exists | `{"exists": true}` / `{"exists": false}` |
| Numeric | `{"numeric": [">=", 50, "<", 200]}` |
| CIDR | `{"cidr": "10.0.0.0/24"}` |

Array values use OR logic. Multiple top-level keys use AND logic. Nested `detail` fields can be matched via nested objects or dot notation.

## API

frizzle speaks two protocols on the same port:

### SDK endpoint — `POST /`

Accepts the real `EventBridgeClient.PutEvents` wire format. Point your SDK's endpoint here.

Header: `X-Amz-Target: AWSEvents.PutEvents`
Content-Type: `application/x-amz-json-1.1`

The `Detail` field must be a JSON string (as the SDK sends it). The `EventBusName` field on each entry selects the bus.

### Curl endpoint — `POST /events/{bus-name}`

Simplified format for manual testing. The bus name is in the URL path. `detail` is a JSON object.

```json
{
  "Entries": [
    {
      "version": "0",
      "id": "evt-001",
      "detail-type": "OrderPlaced",
      "source": "myapp.orders",
      "account": "123456789012",
      "time": "2025-06-15T10:00:00Z",
      "region": "us-east-1",
      "resources": [],
      "detail": { "orderId": "ORD-123" }
    }
  ]
}
```

Response:

```json
{
  "Entries": [{ "EventId": "evt-001" }],
  "FailedEntryCount": 0
}
```

Omitting the bus name (or using `/`) defaults to the first configured bus.

## Target types

| Type | Description |
|---|---|
| `log` | Prints the event to stdout |
| `http` | POSTs the full event JSON to a URL |

## Config location

Per-directory — reads `.frizzle/frizzle.json` from the current working directory. Use `--config` to point at a different file. No global configs.
