// Package a2uiruntime provides a reusable macOS A2UI runtime.
//
// The runtime owns:
//
//   - surface state and data model application
//   - client action transport
//   - client-side function execution
//   - SwiftUI rendering for A2UI v0.9 surfaces
//
// The primary embedding types are:
//
//   - Runtime for surface state, rendering, and support metadata
//   - Client for SSE transport and reconnect behavior
//   - Transport for client-to-server messages
//   - FunctionExecutor for client-only actions such as openUrl
//
// Local extensions remain supported, but are surfaced separately from upstream
// A2UI support through SupportMatrix.
package a2uiruntime
