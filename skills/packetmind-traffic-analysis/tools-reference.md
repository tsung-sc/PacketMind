# PacketMind Tools Quick Reference

| # | Tool | Description | Required Params | Optional Params |
|---|---|---|---|---|
| 1 | `summarize_session` | Session overview: hosts, counts, status/content-type distributions, auth detection, timeline. | — | `session_id` |
| 2 | `classify_requests` | Auto-classify into 10 roles (auth_entry, token_issuance, signed_request, data_query, auth_request, config_fetch, static_resource, redirect, error, other). | — | `session_id`, `host`, `limit` |
| 3 | `trace_flow_sequence` | Chronological flow with cookie/token/value edge detection and anomaly flagging. | — | `session_id`, `host`, `focus_fields`, `max_requests` |
| 4 | `get_request` | Fetch full request/response details by ID (headers, body, timing, TLS, error). | `request_id` | `include_full_body` |
| 5 | `search_requests_by_header` | Find requests matching a header name + value fragment. | `header_name`, `header_value` | `session_id`, `limit` |
| 6 | `search_requests_by_body` | Find requests whose request body contains a text fragment. | `value` | `session_id`, `limit` |
| 7 | `search_requests_by_response` | Find requests whose response body contains a text fragment. | `value` | `session_id`, `limit` |
| 8 | `search_all_fields` | Search headers, body, response body, cookies, and query params simultaneously. | `value` | `session_id`, `limit` |
| 9 | `find_prior_response_sources` | Trace backward: find earlier responses containing a target value. | `before_request_id`, `value` | `session_id`, `limit` |
| 10 | `find_later_request_usages` | Trace forward: find later requests reusing a target value. | `after_request_id`, `value` | `session_id`, `limit` |
| 11 | `trace_value_flow` | End-to-end provenance tracing with confidence scoring. | `request_id`, `field_name`, `location` | `session_id`, `limit` |
| 12 | `test_hypothesis` | Test signature/encoding hypothesis against 3+ requests with expression DSL. | `request_ids`, `target_field`, `target_location`, `hypothesis` | — |
| 13 | `diff_requests` | Field-by-field comparison of two requests (headers, body, cookies, query, timing). | `request_id_a`, `request_id_b` | `fields` |
| 14 | `analyze_encoding` | Detect and decode nested encoding chains (Base64, URL, hex, HTML entities). | `value` | — |
| 15 | `batch_execute` | Execute multiple independent tool calls in parallel. | `calls` | `session_id` |
| 16 | `bash` | Execute shell command in sandboxed workspace (requires user approval). | `command` | `workdir`, `timeout` |
| 17 | *(test_hypothesis returns)* | `alternative_hypotheses` in test_hypothesis result suggests follow-up tests when hypothesis fails. | — | — |

## Search Tool Selection Guide

Use the narrowest tool that fits your need — it produces fewer, more relevant results:

```
Know the header name?         → search_requests_by_header(header_name, header_value)
Looking for a request field?  → search_requests_by_body(value)
Looking for a response field? → search_requests_by_response(value)
No idea which field?          → search_all_fields(value)          ← broadest, slowest
```

## Common Parameter Conventions

- **`session_id`** — Optional on all tools that accept it. When omitted, the
  tool operates on the current active capture session.
- **`limit`** — Integer 1-50 (default varies by tool). Caps result size. Always
  set a limit in `batch_execute` calls to avoid flooding the context window.
- **`request_id`** — Must be a real ID from search results or user-provided
  context. Never fabricate or guess IDs.

## `batch_execute` Usage

The `calls` parameter is a JSON array of tool call objects:
```json
{
  "calls": [
    {"tool": "search_requests_by_header", "args": "{\"header_name\":\"Authorization\",\"header_value\":\"Bearer\"}"},
    {"tool": "search_requests_by_body", "args": "{\"value\":\"access_token\"}"}
  ]
}
```
Each `args` value is a **JSON-encoded string** of the inner tool's parameters.
All calls execute in parallel. Use this when you need multiple independent
searches — do NOT use it for sequential dependencies.

## `bash` Sandbox

- Default working directory: `data/agent-workspace` (relative to PacketMind's
  runtime directory).
- Output truncated at 50000 bytes.
- Timeout: 1-300 seconds (default 60).
- The tool requires user approval at runtime. If the user denies it, explain
  what was blocked and suggest alternative approaches. Do not try to escalate
  or bypass.

## `test_hypothesis` Expression DSL

| Operation | Syntax | Example |
|---|---|---|
| Field extraction | `EXTRACT(location, path)` | `EXTRACT(body, $.data.secret)` |
| Concatenation | `CONCAT(a, b, ...)` | `CONCAT(timestamp, body_raw)` |
| Join with separator | `CONCAT_WITH(sep, a, b)` | `CONCAT_WITH("&", k1, v1)` |
| Case transform | `LOWER(x)`, `UPPER(x)` | `LOWER(CONCAT(a, b))` |
| Hashing | `MD5(x)`, `SHA1(x)`, `SHA256(x)` | `MD5(CONCAT(ts, body))` |
| HMAC | `HMAC_SHA256(data, secret=...)` | `HMAC_SHA256(body, secret=k)` |
| Encoding | `BASE64(x)`, `URLENCODE(x)` | `BASE64(MD5(body))` |
| Literal | `"string"` | `CONCAT("prefix-", ts)` |

**Location values for `EXTRACT` and `test_hypothesis.target_location`:**
`body`, `header`, `query`, `response_body`, `response_header`, `cookie`
