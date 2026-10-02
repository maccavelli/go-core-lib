// Package cli is the canonical self-update command for every program built on
// selfupdate: one set of flags, one stream rule, one set of exit codes
// (docs/decisions/0004-MADR-evolve-selfupdate-api-and-tui-support.md §5).
//
//	<prog> update [--check] [--yes|-y] [--force] [--dry-run] [--json]
//	              [--version vX.Y.Z] [--channel NAME]
//
// Stdout carries machine-readable protocol output and nothing else: without
// --json it stays empty, and under --json it is JSON Lines, one event per
// line, then exactly one {"kind":"result",…} object. Everything else goes to
// stderr. The exit status is 0 when up to date, declined or installed, 10
// when --check finds an update, and 1 on any error.
package cli
