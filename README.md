# frizzle

<p align="center"><img src="frizzle.png" alt="frizzle" width="200"/></p>

A local AWS EventBridge simulator that lets you build and test event-driven systems without needing an AWS account. Point your existing AWS SDK at frizzle, send events, and watch them get routed to your local services — just like the real thing.

## Installation

Three ways to install. Pick whichever fits your workflow — they all produce the same `frizzle` binary.

### Go install

```
go install github.com/LukeOfEarth/frizzle@latest
```

Requires Go 1.22+. Behind the scenes:

1. Go's module proxy finds the latest tagged version by asking `proxy.golang.org`
2. It fetches the module zip (source code) from the proxy
3. It runs `go build` on the `main` package at the module root, linking in all dependencies
4. The resulting binary is placed in `$GOPATH/bin` (usually `~/go/bin`)

This works anywhere Go runs. The binary is compiled for your exact OS and architecture. Version pinning works as usual — `@v0.3.2` for a specific release, `@latest` for the newest.

### Homebrew

```
brew tap LukeOfEarth/tap
brew install frizzle
```

How this works:

1. `brew tap LukeOfEarth/tap` registers `https://github.com/LukeOfEarth/homebrew-tap` as a formula source. This is a one-time setup — Homebrew will check here for `frizzle` from now on.
2. `brew install frizzle` reads the formula (`frizzle.rb`) from that tap repo. The formula tells Homebrew:
   - Which release URL to download from (matched to your OS and CPU architecture)
   - The SHA256 checksum to verify the download
   - How to install (just move the binary into place)
3. Homebrew downloads the pre-built tarball, verifies the checksum, and installs the binary to its prefix (`/opt/homebrew/bin` on Apple Silicon, `/usr/local/bin` on Intel).

The formula is updated manually after each release. If you want a version that hasn't been pushed to the tap yet, use `go install` or grab the binary directly.

### GitHub Releases

Pre-built, statically-linked binaries for every platform:

- `frizzle_{version}_darwin_amd64.tar.gz` — macOS Intel
- `frizzle_{version}_darwin_arm64.tar.gz` — macOS Apple Silicon
- `frizzle_{version}_linux_amd64.tar.gz` — Linux x86_64
- `frizzle_{version}_linux_arm64.tar.gz` — Linux ARM64
- `frizzle_{version}_windows_amd64.zip` — Windows x86_64

Every release is built by [GoReleaser](https://goreleaser.com) via a GitHub Actions workflow that triggers on every `v*` tag. The workflow:
1. Cross-compiles `frizzle` from source for all five targets with `CGO_ENABLED=0` (fully static, no system library dependencies)
2. Packages each binary into a `.tar.gz` (or `.zip` on Windows)
3. Generates a `checksums.txt` with SHA256 hashes
4. Publishes everything as a GitHub Release

No Go toolchain needed on the installing machine — just download, extract, and run.

## Getting started

The fastest way to try frizzle is with the `--forward` flag. This starts a single catch-all bus that logs every event and forwards it to a downstream HTTP endpoint:

```
frizzle start --port 4000 --forward http://localhost:3001/webhook
```

That's it — frizzle is now listening on port 4000. Any event you send will be printed to your terminal and POSTed to `http://localhost:3001/webhook`.

### Pointing your AWS SDK at frizzle

The key idea is simple: tell your EventBridge client to talk to `localhost` instead of AWS. The credentials don't matter (use any dummy values), and frizzle handles the rest.

<details>
<summary><strong>Go SDK v2</strong></summary>

```go
cfg, _ := config.LoadDefaultConfig(ctx,
    config.WithRegion("us-east-1"),
    config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider("test", "test", "")),
)
client := eventbridge.NewFromConfig(cfg, func(o *eventbridge.Options) {
    o.BaseEndpoint = aws.String("http://localhost:4000")
})
```

</details>

<details>
<summary><strong>JavaScript SDK v3</strong></summary>

```js
import { EventBridgeClient, PutEventsCommand } from "@aws-sdk/client-eventbridge";

const client = new EventBridgeClient({
  region: "us-east-1",
  endpoint: "http://localhost:4000",
  credentials: { accessKeyId: "test", secretAccessKey: "test" },
});
```

</details>

<details>
<summary><strong>Python (boto3)</strong></summary>

```python
import boto3

client = boto3.client("events",
    region_name="us-east-1",
    endpoint_url="http://localhost:4000",
    aws_access_key_id="test",
    aws_secret_access_key="test",
)
```

</details>

<details>
<summary><strong>Java SDK v2</strong></summary>

```java
EventBridgeClient client = EventBridgeClient.builder()
    .region(Region.US_EAST_1)
    .endpointOverride(URI.create("http://localhost:4000"))
    .credentialsProvider(StaticCredentialsProvider.create(
        AwsBasicCredentials.create("test", "test")))
    .build();
```

</details>

<details>
<summary><strong>C# (.NET SDK)</strong></summary>

```csharp
using Amazon.EventBridge;
using Amazon.Runtime;

var client = new AmazonEventBridgeClient(
    new BasicAWSCredentials("test", "test"),
    new AmazonEventBridgeConfig
    {
        ServiceURL = "http://localhost:4000",
        AuthenticationRegion = "us-east-1"
    });
```

</details>

<details>
<summary><strong>Rust SDK</strong></summary>

```rust
use aws_sdk_eventbridge::{config::Region, Client, Config};
use aws_credential_types::Credentials;

let creds = Credentials::new("test", "test", None, None, "static");
let config = Config::builder()
    .region(Region::new("us-east-1"))
    .endpoint_url("http://localhost:4000")
    .credentials_provider(creds)
    .build();
let client = Client::from_conf(config);
```

</details>

Once configured, your application's `PutEvents` calls will go to frizzle instead of AWS — no other code changes needed.

### Quick test with curl

You don't need an SDK to try things out. frizzle has a simplified HTTP endpoint designed for manual testing:

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

## Configuration file

For more advanced setups — multiple buses, event routing rules, and pattern matching — you'll want a config file.

Generate one with `frizzle init`:

```
frizzle init                # creates a full example config with comments
frizzle init --simple       # creates a minimal single-bus config
frizzle start               # automatically reads .frizzle/frizzle.json from the current directory
```

### Config structure

The config file lives at `.frizzle/frizzle.json` in your project directory. Here's what a typical one looks like:

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

**How rules work:** Rules are evaluated top-to-bottom. When an event matches a rule's pattern, all of that rule's targets fire (fan-out). If you want a catch-all to log unmatched events, add a rule with an empty `pattern: {}` at the bottom — it matches everything.

You can also point to a config file in a different location with `--config path/to/config.json`.

## Pattern matching

frizzle supports the full EventBridge pattern syntax, so the rules you write locally will behave the same way they do in production.

| Operator | Example | What it does |
|---|---|---|
| Exact match | `"source": ["myapp.orders"]` | Matches the exact string value |
| Prefix | `{"prefix": "myapp.orders"}` | Matches values starting with the given string |
| Suffix | `{"suffix": ".pdf"}` | Matches values ending with the given string |
| Wildcard | `{"wildcard": "report_*.pdf"}` | Matches using `*` as a wildcard |
| Case-insensitive | `{"equals-ignore-case": "orderplaced"}` | Matches regardless of case |
| Anything-but | `{"anything-but": ["val1", "val2"]}` | Matches any value *except* those listed |
| Anything-but (prefix) | `{"anything-but": {"prefix": "prod."}}` | Matches values that *don't* start with the given prefix |
| Exists | `{"exists": true}` / `{"exists": false}` | Checks whether a field is present (or absent) |
| Numeric | `{"numeric": [">=", 50, "<", 200]}` | Compares numeric values with range operators |
| CIDR | `{"cidr": "10.0.0.0/24"}` | Matches IP addresses within a CIDR block |

When you list multiple values in an array, they're combined with **OR** logic (the event matches if *any* value matches). When you have multiple top-level keys in a pattern, they're combined with **AND** logic (the event must match *all* of them). You can match nested fields inside `detail` using nested objects or dot notation.

## API

frizzle serves two protocols on the same port, so you can use it with both real AWS SDKs and manual testing tools without any extra configuration.

### SDK endpoint — `POST /`

This is the endpoint your AWS SDK talks to. It accepts the real `PutEvents` wire format with the standard AWS headers:

- Header: `X-Amz-Target: AWSEvents.PutEvents`
- Content-Type: `application/x-amz-json-1.1`

The `Detail` field must be a JSON string (which is how the SDK sends it). Each entry's `EventBusName` field determines which bus it's routed to.

### Curl-friendly endpoint — `POST /events/{bus-name}`

A simplified format designed for manual testing. The bus name goes in the URL path, and the `detail` field is a regular JSON object (not a string):

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

Both endpoints return the same response format:

```json
{
  "Entries": [{ "EventId": "evt-001" }],
  "FailedEntryCount": 0
}
```

If you omit the bus name (or just POST to `/`), the event goes to the first configured bus.

## Target types

| Type | What it does |
|---|---|
| `log` | Prints the full event to stdout — useful for debugging |
| `http` | POSTs the event as JSON to a URL — use this to wire up your local services |
