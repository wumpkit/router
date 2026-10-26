# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [1.0.0] - 2026-10-26

### Added
- **Type-Defined Commands**: First-class command definition using idiomatic Go types and structs implementing command interfaces (`ChatCommand`, `ButtonCommand`, `SelectMenuCommand`, `ModalCommand`, `UserContextMenuCommand`, `MessageContextMenuCommand`).
- **Strongly Typed Contexts**: Each component and context-menu command accepts its concrete context directly (`*ButtonCommandContext`, `*SelectMenuCommandContext`, `*ModalCommandContext`, `*UserContextMenuCommandContext`, `*MessageContextMenuCommandContext`) eliminating manual type assertions.
- **Zero-Allocation Routing Fast Paths**:
  - `tokenizePath` in component pattern matcher now uses string slicing without heap allocations for ASCII paths.
  - `ParseTokens` in message tokenizer includes an early zero-allocation pass for positional-only commands.
- **Convenient Response Helpers**: Added `ReplyEmbed`, `ReplyEphemeral`, `Followup`, `Edit`, and `Delete` on `Context` and `MessageCommandContext`.
- **Custom Argument Parsers**: Support custom parsers via `ArgDef.Parser func(string) (any, error)`.
- **Router Lifecycle Hooks**: Added `Before` and `After` execution pipeline hooks for telemetry, metric tracking, and audit logging.
- **Dynamic Prefix Support**: Added `WithDynamicPrefixFunc` for per-guild or per-user dynamic prefixes alongside static prefixes and mentions.
- **Pluggable Cooldown Stores**: Added `CooldownStore` interface and `middleware.CooldownWithStore` enabling external stores (e.g. Redis).
- **Auto-Sync Application Commands**: Added `r.AutoSync(appID, guildIDs...)` for automatic command synchronization on bot ready event.
- **Enterprise Community & Security Governance**: Contributor Covenant 2.1 code of conduct, security disclosure policy, issue/PR templates, and comprehensive CI/CD pipelines.

### Changed
- Bumped `github.com/wumpkit-go/wumpkit` dependency to `v1.0.0`.
- Standardized toolchain and matrix tests on Go 1.23, 1.24, and 1.25.

## [0.1.0] - 2026-08-27

### Added
- Initial release of Wumpkit Router.
- Support for chat commands with prefix tokenization and flags.
- Support for slash commands, user context menu commands, and message context menu commands.
- Pattern matching for button, select menu, and modal component interactions.
- GCRA in-memory cooldown rate limiter middleware.
- Structured execution logging and panic recovery middleware.
