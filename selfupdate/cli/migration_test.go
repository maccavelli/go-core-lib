package cli

import (
	"strconv"
	"testing"
)

// migrationProduct is the consumer the migration proof moves to Command
// (0004-PLAN-v1-4-0-command-surface.md Step 7).
const migrationProduct = "prepare-commit-msg"

// TestMigrationFixture writes, or checks, what `prepare-commit-msg update
// --check` must print once it calls Command: testdata/migration/<scenario>
// .stdout, .stderr and .code. A scratch copy of prepare-commit-msg is
// compared with these files byte for byte (A23).
func TestMigrationFixture(t *testing.T) {
	for _, sc := range scenarios {
		switch sc.name {
		case "up-to-date", "available", "failed":
		default:
			continue
		}
		t.Run(sc.name, func(t *testing.T) {
			out := sc.command(t, migrationProduct, false)
			goldenIn(t, "migration", sc.name+".stdout", out.stdout)
			goldenIn(t, "migration", sc.name+".stderr", out.stderr)
			goldenIn(t, "migration", sc.name+".code", strconv.Itoa(out.code)+"\n")
		})
	}
}
