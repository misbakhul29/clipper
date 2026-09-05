# Project Rules & Customizations (`clipper`)

## Git Commit & Versioning Conventions

Every time a new feature is implemented or a major bug is fixed in this repository, follow these rules:

1. **Commit Message Format**:
   - Use Conventional Commits with a **Naruto Jutsu** reference theme:
     - Features: `feat(jutsu): <Jutsu Name> vX.Y.Z - <Short Description> 🌀⚡`
     - Fixes: `fix(jutsu): <Jutsu Name> vX.Y.Z - <Short Description> 🛠️`
   - Examples:
     - `feat(jutsu): Chidori Precision Cut v1.0.0 - Automated Video Clipper & Shorts Engine ⚡`
     - `feat(jutsu): Rasenshuriken Multi-Tasking v1.1.0 - Concurrency Worker Pool & Silence Detection 🌀`

2. **Semantic Versioning & Git Tags**:
   - Always update Semantic Versioning (`vX.Y.Z`):
     - `MINOR` bump (`v1.1.0` -> `v1.2.0`) for new feature implementations.
     - `PATCH` bump (`v1.1.0` -> `v1.1.1`) for major bug fixes.
     - `MAJOR` bump (`v1.0.0` -> `v2.0.0`) for breaking changes.
   - Always create an annotated Git tag matching the version:
     `git tag -a vX.Y.Z -m "vX.Y.Z: <Summary>"`

3. **UI & Design Rules (Strict No-Emoji Policy)**:
   - **Never use emojis in any UI**: All Web UI headers, labels, buttons, cards, modals, select dropdown options, alerts, and CLI interactive wizard prompts/logs must NEVER contain emojis.
   - Use clean, modern SVG icons and professional typography instead of emojis in all user interfaces.
   - Emojis are strictly prohibited in the UI layer.

4. **Workflow Rule**:
   - Proactively assist the user with `git add`, `git commit`, and `git tag` upon finishing feature implementations or major bug fixes.
