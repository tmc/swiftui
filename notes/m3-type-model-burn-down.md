# M3 Type-Model Burn-Down

This note is the implementation plan for the `policy_*` backlog families in the SwiftUI report. It stays out of runtime-model and emitter work.

Current report families to burn down:

- `policy_unresolved_type`: `398`
- `policy_generic_specialization`: `196`
- `policy_nested_type`: `72`
- `policy_collection`: `38`

The goal is to turn these from ad hoc skip reasons into reusable recognizers and small policy rules in `appledocs/internal/swiftbridge`, without widening the curated SwiftUI catalog.

## Working Order

1. Improve unresolved-type recognition first, because it feeds the other three buckets.
2. Add a specialization registry for repeated generic shapes.
3. Tighten nested-type flattening rules.
4. Add typed collection adapters only for the few homogeneous shapes that recur.

## 1. Unresolved Type

This bucket is mostly a parser and type-resolution problem, not a generator problem.

High-leverage changes:

- Reuse and extend `unknownTypeNeedsSwiftUIPolicy` in [`type_policy.go`](/Volumes/tmc/go/src/github.com/tmc/appledocs/internal/swiftbridge/type_policy.go) so it distinguishes:
  - true unknown types
  - protocol-shaped names
  - placeholder/generic member names
  - nested names with generic parents
- Add a normalization step in [`parser.go`](/Volumes/tmc/go/src/github.com/tmc/appledocs/internal/swiftbridge/parser.go) so dotted members are preserved as structured type paths instead of falling through to a generic "unresolved" bucket.
- Keep protocol-shaped SwiftUI types out of the "unknown" bucket when they are already recognized as protocol-like and can be classified separately.

Reusable recognizers already in place:

- `isLikelyProtocolShapedName`
- `isLikelyGenericPlaceholderName`
- `nestedTypeHasGenericParent`
- `signatureHasUnresolvedTypes`

What to add:

- A helper that canonicalizes dotted names before policy classification.
- Tests for SwiftUI-specific names such as `View`, `Shape`, `Scene`, and nested members like `Namespace.ID`.

## 2. Generic Specialization

This bucket is the best place to add reusable overlays instead of exposing open generics.

High-leverage changes:

- Add a small specialization registry keyed by nominal type plus type-argument shape.
- Reuse `signatureHasConcreteSpecializationCandidate` and `typeCanParticipateInConcreteSpecialization` as the admission test for a candidate overlay.
- Promote only closed, concrete shapes that are already repeated in the report, rather than inventing a generic public API.

Good first shapes:

- `Binding<Bool>`
- `Binding<String>`
- `Binding<Date>`
- `Set<Date>`
- timer-backed `ProgressView` shapes
- repeated `Picker` and `NavigationSplitView` concrete forms

What to add:

- A registry that maps a generic SwiftUI nominal to one or more concrete overlays.
- Tests that verify concrete overlays are recognized consistently across inits, methods, and properties.

## 3. Nested Type

This bucket should be handled by canonicalizing concrete nested types, while still rejecting nested generics.

High-leverage changes:

- Reuse `extractNestedTypeName` and `nestedTypeHasGenericParent` from [`type_policy.go`](/Volumes/tmc/go/src/github.com/tmc/appledocs/internal/swiftbridge/type_policy.go) and [`bridgeability.go`](/Volumes/tmc/go/src/github.com/tmc/appledocs/internal/swiftbridge/bridgeability.go).
- Make nested-type normalization produce stable Go names for concrete nested members.
- Keep nested types with generic parents in the hard-error path instead of trying to flatten them.

What to add:

- A dedicated nested-type canonicalizer that runs before final policy classification.
- Tests for concrete nested members versus nested members under generic parents.

## 4. Collection

This bucket should stay narrow. It is for typed adapters, not for a broad "any array" bridge.

High-leverage changes:

- Reuse `signatureHasTypedCollectionCandidate` and `typeCanUseTypedCollectionElement`.
- Keep typed collection support only for homogeneous element families that already show up repeatedly in SwiftUI docs.
- Do not use collection bridging as a back door for generic specialization or unresolved type cleanup.

Good first shapes:

- arrays of simple value types
- dictionaries with fixed value shapes
- gradient stops and point/color collections
- grid and paste-related content-type collections when the element type is known

What to add:

- A typed collection adapter table, not a generic collection policy.
- Tests that prove collection bridging stays limited to supported element shapes.

## Highest-Leverage Implementation Sequence

1. Add a canonical dotted-type normalization pass and tighten unresolved-type classification.
2. Build the specialization registry on top of the existing concrete-specialization recognizers.
3. Add nested-type canonicalization and keep generic parents rejected.
4. Add the narrow collection adapter set last.

## Exit Criteria

The M3 work is done when the report can separate these cases cleanly:

- unresolved or protocol-shaped types that still need parser/policy work
- concrete generic shapes that are ready for specialization overlays
- concrete nested types that can be flattened safely
- typed collection shapes that have a clear adapter

At that point, the remaining bucket counts should shrink for the right reasons, and the report should no longer treat every difficult type as the same kind of backlog.
