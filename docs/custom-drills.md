# Custom drills

A custom drill is a folder in `drills/<id>/` under the [config directory](configuration.md) with a `drill.json` and any other files the drill needs. Resources are copied into the sprint workspace; declared student files start empty or absent, and the root `solution/` reference directory stays outside the workspace. Create one with `codedrill new`, or write the JSON and run `codedrill import drill.json`.

A local drill with a built-in ID replaces that built-in in the menu. This includes
existing `exam05-*` drills: upgrading does not rewrite, migrate, or delete them.
Use a separate `XDG_CONFIG_HOME` to try the built-ins without your local overrides.
Exact drill IDs take precedence over matching titles in CLI lookups.

```json
{
  "title": "ft_putchar",
  "category": "42",
  "language": "c",
  "difficulty": "easy",
  "description": "Write a function that prints one character.",
  "entry_file": "ft_putchar.c",
  "starter_code": "",
  "test_command": "gcc -Wall -Wextra -Werror -c ft_putchar.c",
  "auto_submit": false
}
```

- `codedrill import` requires `title` and `entry_file`; `entry_file` must be a relative path.
- When loading a drill folder, `id` defaults to the folder name. On import, a missing `id` is derived from the title.
- `category` can name a subcategory after a slash, e.g. `"42/Exam 05"`. The menu shows these drills in a collapsible `Exam 05` group under the 42 tab. Legacy `42 Exam` and `42 Exam NN` categories are read as `42` and `42/Exam NN`.
- If `test_command` is empty, codedrill picks one from the file extension (C, C++, Go, Rust, Python, JavaScript, TypeScript, shell).
- In Zellij, `auto_submit: true` submits the run when the editor closes.
- `extra_files` is a map of file name to contents, written into the workspace.
- `entry_file` is always student-owned and starts at zero bytes, even if a directory attachment, `extra_files`, or legacy `starter_code` supplies its contents.
- `student_files` lists additional required student paths, for example `["set.cpp", "set.hpp"]`. These files also start at zero bytes. `optional_student_files` lists student paths that should be absent until created. Ownership lists override attached or inline contents; they never change the original drill files.
- Unlisted attachments and `extra_files` remain provided resources for compatibility. Explicitly classify student headers and implementations; extensions do not determine ownership. The same initialization applies in every mode.
- `starter_code` remains readable and exportable for older definitions, but no longer seeds a workspace. JSON export includes ownership lists and `extra_files`; physical directory attachments still need to be transferred separately.
- Test evaluation and `r` reject workspaces where all declared student files are empty or whitespace-only, including with explicit compile-only commands. Provided resources do not count as student work. Once work exists, the configured command runs as before and must test the exercise correctly. Copy/Recall reference gates are unchanged.

Resuming an evaluation reopens the current work. Retrying from the results screen creates a fresh empty workspace. Reset applies only to the declared student paths, leaving resources and other files intact.

`codedrill rm` on a custom drill deletes its folder and runs. On a built-in drill it only hides it; the ID is kept in `hidden.json` in the config directory. Use `codedrill restore <id>` or `codedrill restore all` to bring it back.
