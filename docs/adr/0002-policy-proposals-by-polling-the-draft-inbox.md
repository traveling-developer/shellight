# Policy proposals come from polling the draft inbox

shellight reads pending policy proposals by calling `Policy().GetDraft` for each
sandbox in every poll round, instead of following each sandbox's log stream
(`WatchSandbox` with `follow_logs`) and parsing OCSF denial lines as the earlier
spike did. The Go SDK does not expose the `DraftPolicyUpdate` push, and the
proposal (not the raw denial) is what the user approves or rejects, so polling
the inbox gives the right object directly at the cost of up to one poll
interval (5 s) of latency.

## Considered Options

- Log stream per sandbox: denials arrive in under a second, but they are
  events, not proposals, need OCSF parsing, bundling of DNS + connect pairs,
  cursor handling and reconnects.
