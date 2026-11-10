# Exam 05 practice collection

The binary embeds five exercises under `42/Exam 05`, including their statements,
student-file ownership, resources, and references:

| ID | Required student files | Behavioral checks |
| --- | --- | --- |
| `exam05-bigint` | `bigint.cpp`, `bigint.hpp` | Decimal addition, comparisons, copying, increment, digit shifts, and values beyond machine precision |
| `exam05-bsq` | `bsq.c` | Largest-square results against a brute-force oracle, tie selection, header forms, malformed maps, stdin, and multiple files |
| `exam05-life` | `life.c` | Drawing, clamped edges, finite-board evolution against a separate oracle, and exact output |
| `exam05-polyset` | Six `searchable_array_bag`, `searchable_tree_bag`, and `set` source/header files | Membership, duplicates, const access, populated copies, assignment, clearing, destruction, and wrapper forwarding |
| `exam05-vect2` | `vect2.cpp`, `vect2.hpp` | Independent coordinates and copies, arithmetic, equality, indexing, operator return identity, and stream formatting |

The C exercises permit optional `bsq.h` and `life.h`, respectively. Their commands
compile all student-created `*.c` files. The C++ commands compile the required
implementation files with the Codedrill test main. No compiler standard flag is
forced; the supplied materials need C99 and C++11 support.

## Local data and practice modes

Resources are embedded with Go's `embed` package, so a relocated executable or a
Nix installation needs no source checkout. Required student files begin empty
and optional files begin absent in every mode. References remain in the binary,
outside the student workspace. Mode visibility is a practice feature, not a
security boundary against someone inspecting their own executable.

Local drills with matching IDs still replace built-ins. Local reference
attachments still replace the entire selected reference. No user configuration,
history, preferences, or local exercise files are migrated or overwritten.

Study and Exam require passing tests. Copy and Recall require exact normalized
reference matches; tests still run but do not decide their default pass result.
Custom settings can require both. A shorter correct implementation can therefore
pass Study or Exam and fail Copy or Recall.

To inspect only the built-in collection without changing your usual config:

```sh
XDG_CONFIG_HOME="$(mktemp -d)" codedrill list 42
```

## Third-party text and code

The local collection inspected for this integration contained `bigint`, `bsq`,
`life`, `polyset`, and `vect2` manifests, test scripts, resources, and attached
references under `~/.config/codedrill/`. Additional subjects and implementations
were under the local `EXAM5/exam05/subjects`, `EXAM5/examclaude`, and
`EXAM5/exam_repos` collections, including the `life_zero`, `alt_xy`,
`alt_littleendian`, `alt_grid`, `alt_2dmatrix`, and `ocf_version` variants.

No LICENSE/COPYING file, explicit redistribution grant, or express prohibition
was found in that local source collection. Some resources carry student
attribution; some statements are reconstructions of school subjects. Permissions
for verbatim copying those particular files remain unestablished. This does not
exclude the exercises or their required interfaces and behavior from Codedrill.

The bundled descriptions, tests, references, and polyset support code are new
implementations of those exercises. No third-party subject text or source files
are imported. Any future verbatim import needs a separate permissions review.

Everything under `internal/exercises/exam05/` was written for Codedrill in this
change and is covered by the repository's MIT license. The existing local
collection was used to identify exercise behavior and interfaces, not as source
text to publish. This is not a claim of official school provenance or endorsement.

## Differences and remaining limits

- These statements and tests preserve the requirements found in the inspected
  subjects and required mains. They do not reproduce the private school grader.
  The local inventory does not establish the complete pool for every school,
  date, or exam instance.
- `life` prints the letter `O`. The observed digit `0` variant is not a separate
  built-in drill. Existing local variants remain available.
- Polyset's support code is rewritten with the local subject resources' public
  and protected APIs and observable output. Its tree discards duplicates, omits
  zero from printing, and emits the existing allocation/destruction messages.
  Those supplied-resource quirks are preserved, not imposed as student logic.
  The base destructors are not virtual, so tests destroy concrete objects.
- The two searchable polyset classes are asked to declare all four canonical
  special members. Runtime tests establish behavior, not whether those members
  were handwritten. The reference's noncopyable set wrapper is an implementation
  choice, not a test requirement. The original subject requests canonical form
  for all classes but does not specify the wrapper's default/copy behavior;
  the supplied main does not exercise those operations.
- Allowed C function lists are documented and followed by the references, but
  are not enforced by the tests. Tests also cannot prove absence of every leak,
  undefined operation, allocation failure, or excessive resource use.
- Supplied workspace resources are editable. Altering a harness can invalidate
  the meaning of a pass; Codedrill does not authenticate a school submission.

## Verify the collection

Run the full fresh-config exercise check:

```sh
go test ./internal/runner -run '^TestExam05FreshInstall$' -count=1 -v
```

It runs each embedded reference through its actual drill command, checks empty
workspaces and mode policies, compares exact and equivalent alternative sources,
and proves a compiling behavioral mutation fails each exercise. Local override
and reference-view tests run in the normal `go test ./...` suite. Nix builds also
run these checks with compilers and Python supplied as check dependencies.
