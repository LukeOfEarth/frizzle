---
name: frizzle
description: Local AWS EventBridge simulator for testing event-driven systems. Use when the user mentions event buses, EventBridge, frizzle, local event testing, event-driven architecture testing, or needs to wire up event publishing/consuming for local development. Also use when setting up a local dev environment for a project that publishes or consumes AWS EventBridge events — detect PutEvents/EventBridgeClient calls in the codebase and proactively offer to configure frizzle for them.
---

# Frizzle — Local EventBridge Simulator

frizzle is a CLI tool that runs on `localhost` and intercepts EventBridge `PutEvents` calls from the AWS SDK. It matches events against rules, logs them, and forwards them to configured downstream HTTP targets. No AWS account needed.

## When to use this skill

Use frizzle whenever the user needs to:

- Test EventBridge event publishing/consuming locally
- Wire up event-driven services during local development
- Debug event routing and pattern matching without deploying to AWS
- Set up a local dev environment for a project that uses EventBridge

**Trigger detection in codebase:** Before offering frizzle, scan the project for:
- `EventBridgeClient`, `PutEventsCommand`, `PutEvents` in code
- `eventbridge.NewFromConfig`, `eventbridge.New`
- `boto3.client("events"` or `boto3.client('events'`
- `@aws-sdk/client-eventbridge` in `package.json`
- `aws-sdk-go-v2/service/eventbridge` in `go.mod`

Also watch for mention of event types like `OrderPlaced`, `UserCreated`, etc. — these are event detail-types that frizzle can route.

## Workflow: zero-questions setup

Follow this sequence. Do not ask the user questions — derive answers from the codebase.

### Step 1: Inventory events from the codebase

Search the project for every `PutEvents` / `PutEventsCommand` call. For each one, record:

- **Event source** — the `Source` field value (e.g. `myapp.orders`)
- **Detail type** — the `DetailType` field value (e.g. `OrderPlaced`)
- **Bus name** — the `EventBusName` field value (e.g. `default`)
- **Detail shape** — what keys/values go into the `Detail` JSON

If a field uses a variable, trace it back to a constant or configuration. If you can't determine the exact value, use a reasonable default and note the assumption.

### Step 2: Check for downstream consumers

Look for where these events are consumed:

- Lambda handlers, HTTP endpoints, SQS queues that receive the events
- Target URLs for the actual EventBridge rules (often in CDK, CloudFormation, or Terraform)
- Local service endpoints that handle these events in dev

Each consumer becomes an HTTP target in the frizzle config.

### Step 3: Generate the frizzle config

Run `frizzle init --simple` to create the scaffold, then write the actual config. The config lives at `.frizzle/frizzle.json`.

If `frizzle` is not installed, install it first: `go install github.com/LukeOfEarth/frizzle@latest` or `brew tap LukeOfEarth/tap && brew install frizzle`.

The config maps each event bus to its rules and targets. For the full schema, read `references/config-reference.md`.

Basic structure:
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
              "timeout": "5s"
            }
          ]
        },
        {
          "name": "catch-all",
          "pattern": {},
          "targets": [{ "type": "log" }]
        }
      ]
    }
  ]
}
```

**Rules checklist:**
- Add one rule per event group (same source + detail-type combinations)
- Add a catch-all rule at the bottom with empty `pattern: {}` to log unmatched events
- For each downstream consumer, add an `http` target with the correct URL
- Always include a `log` target on each rule for debugging visibility
- Use the `timeout` field on HTTP targets to avoid hanging (default 10s)

### Step 4: Configure the project's SDK

Add a local endpoint override to the project's EventBridge client. Use the port from the config (default 4000).

**Go SDK v2:**
```go
client := eventbridge.NewFromConfig(cfg, func(o *eventbridge.Options) {
    o.BaseEndpoint = aws.String("http://localhost:4000")
})
```

**JavaScript/TypeScript SDK v3:**
```ts
const client = new EventBridgeClient({
  region: "us-east-1",
  endpoint: "http://localhost:4000",
  credentials: { accessKeyId: "test", secretAccessKey: "test" },
});
```

**Python boto3:**
```python
client = boto3.client("events",
    region_name="us-east-1",
    endpoint_url="http://localhost:4000",
    aws_access_key_id="test",
    aws_secret_access_key="test",
)
```

**Java SDK v2:**
```java
EventBridgeClient client = EventBridgeClient.builder()
    .region(Region.US_EAST_1)
    .endpointOverride(URI.create("http://localhost:4000"))
    .credentialsProvider(StaticCredentialsProvider.create(
        AwsBasicCredentials.create("test", "test")))
    .build();
```

**Rust SDK:**
```rust
let creds = Credentials::new("test", "test", None, None, "static");
let config = Config::builder()
    .region(Region::new("us-east-1"))
    .endpoint_url("http://localhost:4000")
    .credentials_provider(creds)
    .build();
let client = Client::from_conf(config);
```

**Making it toggleable:** Wrap the endpoint override in an environment variable so the user can switch between local and real AWS:

```go
endpoint := os.Getenv("EVENTBUS_ENDPOINT")
if endpoint != "" {
    // set BaseEndpoint
}
```

### Step 5: Start frizzle and verify

Run `frizzle start` from the project directory. It reads `.frizzle/frizzle.json` automatically.

Verify by either:
- Running the project's event-publishing code path (e.g., hit an API endpoint that publishes an event)
- `curl -X POST http://localhost:4000/events/default -H "Content-Type: application/json" -d '{"Entries":[{"source":"myapp.orders","detail-type":"OrderPlaced","detail":{"orderId":"test-1"}}]}'`

Check the frizzle terminal for the logged event, matched rule, and forwarding result.

## Pattern matching reference

frizzle supports the full EventBridge pattern syntax:

| Operator | JSON | What it matches |
|---|---|---|
| Exact string | `"source": ["myapp.orders"]` | Exact string match |
| Prefix | `{"prefix": "myapp.orders"}` | Starts with string |
| Suffix | `{"suffix": ".pdf"}` | Ends with string |
| Wildcard | `{"wildcard": "report_*.pdf"}` | Glob pattern |
| Case-insensitive | `{"equals-ignore-case": "orderplaced"}` | Case-insensitive match |
| Anything-but | `{"anything-but": ["v1", "v2"]}` | Any except listed values |
| Anything-but prefix | `{"anything-but": {"prefix": "prod."}}` | Doesn't start with prefix |
| Exists | `{"exists": true}` / `{"exists": false}` | Field presence |
| Numeric | `{"numeric": [">=", 50, "<", 200]}` | Numeric range |
| CIDR | `{"cidr": "10.0.0.0/24"}` | IP in CIDR block |

Array values = OR logic. Multiple top-level keys = AND logic.

## Quick start without config

For one-off testing or projects with a single downstream, skip the config file:

```
frizzle start --port 4000 --forward http://localhost:3001/webhook
```

This creates a single bus with a catch-all rule that logs everything and forwards to the given URL.

## Troubleshooting

**"no .frizzle/frizzle.json found"**: Run `frizzle init --simple` first, or use `--forward` for config-free mode.

**"connection refused" on forward**: The downstream service isn't running. Start it or remove that target from the config.

**404 from SDK**: The SDK isn't pointing at frizzle. Verify the endpoint override is set to `http://localhost:<port>` (not `https://`).

**Events not matching rules**: Check the pattern syntax. Array values are OR'd, top-level keys are AND'd. Use the catch-all rule to verify events are reaching frizzle at all.
