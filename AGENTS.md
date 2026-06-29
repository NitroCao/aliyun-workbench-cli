## Project Guidance

- Do not export identifiers just because they are convenient to name from tests or nearby code. Export only package APIs that are actually used across package boundaries; keep protocol details, helper types, internal errors, and connection wrappers unexported.
- When a feature involves several pieces of state and many cooperating functions, group it around a small object with methods instead of passing the same arguments through a long chain of free functions. Keep `context.Context` as a method parameter rather than storing it on the struct.
- Avoid duplicating payload construction logic. If two API/WebSocket messages share the same fields, extract a helper that builds the shared part and have each caller add only its message-specific fields.
- Prefer efficient data flow. Do not convert between `[]byte` and `string` unless the conversion is required for semantics such as JSON field names, command names, or UTF-8/UTF-16 length calculation. Terminal and WebSocket payloads should stay as `[]byte` where possible.
- Keep protocol parsers honest about the protocol. Workbench frame lengths are JavaScript string lengths, not byte lengths, and message separators are valid only after a length-prefixed segment has been parsed.
- Use maintained libraries for standard protocols instead of hand-rolling them. For WebSocket behavior, use the project WebSocket library rather than manually constructing HTTP upgrade requests or frames.
- Keep debug logging diagnostic rather than exhaustive. Log enough structure to troubleshoot requests and WebSocket messages, but do not dump reusable credentials or raw sensitive payloads.
