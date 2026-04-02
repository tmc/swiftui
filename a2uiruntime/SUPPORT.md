# A2UI Runtime Support

Upstream A2UI `v0.9` support in this runtime is centered on the basic catalog
and explicit client behavior.

Supported upstream behavior:

- surface lifecycle: create, update components, update data model, delete
- dynamic values: literals, bindings, and client-side functions
- builtins: `and`, `or`, `not`, `required`, `length`, `numeric`, `email`, `regex`, `formatString`, `formatNumber`, `formatCurrency`, `formatDate`, `pluralize`
- child composition: static child lists and `ChildTemplate`
- validation: `ValidationRegexp` and component `Checks`
- client actions: server events and `openUrl`
- theme metadata: `PrimaryColor`, `AgentDisplayName`, `IconURL`

Local extension support:

- `Progress` component
- `Padding`
- `Spacing`
- `Strikethrough`

Renderer policy notes:

- modal chrome is renderer-defined on macOS because A2UI modals only carry trigger and content IDs
- media playback follows runtime media policy; remote playback can be disabled
