# Snapshot Testing Workflow

This project uses **snapshot testing** (similar to Rust's [insta](https://docs.rs/insta/latest/insta/)) to ensure parser output stays correct.

## How It Works

1. **Tests always regenerate fresh output** from parsing
2. **Compare fresh vs committed snapshots** (golden files)
3. **If different**: Write `.new` files and FAIL
4. **Developer reviews** the changes
5. **Accept or reject** the new snapshots
6. **Git diff review** before committing

## The Safety Net

The key safety is **manual review**:

```bash
# You change the parser
go test  # ❌ FAILS - snapshots differ

# .new files are created automatically
# Review what changed:
make snapshot-review

# If changes look CORRECT:
make snapshot-accept

# If changes look WRONG:
make snapshot-reject
# Then fix your parser bug and re-run tests
```

## Why This Works Better Than `-update` Flags

**Problem with simple `-update`:**
- Easy to blindly run `go test -update` without looking
- Might accidentally commit broken snapshots
- No forced review step

**Our approach (insta-inspired):**
- `.new` files are written automatically (not committed - in .gitignore)
- Test FAILS and shows you the diff
- Clear review workflow with `make snapshot-review`
- Explicit accept/reject commands
- Git diff shows final changes before commit

## Workflow

### Normal Development

```bash
# Run tests
go test

# ✅ All pass - no action needed
```

### After Parser Changes

```bash
# Run tests
go test

# ❌ Tests FAIL - snapshots differ
# .new files created automatically

# Review the changes
make snapshot-review

# You see:
# === testdata/form4/wave/expected.json ===
# --- expected.json
# +++ expected.json.new
# - "shares": "60000",
# + "shares": 60000,

# If this looks CORRECT (numbers instead of strings):
make snapshot-accept

# This runs: go test -update
# Removes .new files
# Updates golden files

# Review with git
git diff testdata/

# If good, commit
git add testdata/
git commit -m "Update snapshots for numeric types"
```

### If Snapshots Are Wrong

```bash
go test  # ❌ FAILS

make snapshot-review
# Shows:
# - "shares": 60000,
# + "shares": 0,      # BUG!

# Reject the bad snapshots
make snapshot-reject

# Fix the parser bug
vim form4_output.go

# Re-run tests
go test  # ✅ PASS (now matches committed snapshots)
```

## Commands

```bash
make snapshot-review   # Show diffs for all .new files
make snapshot-accept   # Accept all changes (runs go test -update)
make snapshot-reject   # Reject changes (delete .new files)
```

## What Gets Committed

✅ **Committed:**
- `testdata/**/ expected.json` (golden files)
- Updated after review and acceptance

❌ **NOT committed (.gitignore):**
- `testdata/**/*.new` (pending changes)
- Created during test failures
- Deleted after accept/reject

## CI/CD Behavior

In CI, tests will **FAIL** if:
1. Parser output doesn't match committed snapshots
2. `.new` files are accidentally committed

This ensures:
- Snapshots are always up to date
- All changes are reviewed before merging
- No accidental regressions

## Limitations

**Snapshot tests are NOT correctness tests!**

They only ensure "output hasn't changed unexpectedly."

They DON'T ensure "output is correct."

### Complementary Testing

Add **invariant checks** for correctness:

```go
// In verifyHelperMethods():
for _, txn := range transactions {
    shares, _ := txn.Shares.Float64()

    // Sanity checks
    if txn.TransactionCode == "S" && shares == 0 {
        t.Error("Sale with 0 shares - parser bug!")
    }
    if shares < 0 {
        t.Error("Negative shares - parser bug!")
    }
}
```

These catch bugs that snapshots can't (e.g., parser always returning 0).

## Best Practices

1. **Always review diffs** before accepting snapshots
2. **Reject suspicious changes** and investigate
3. **Add invariant tests** for critical logic
4. **Use git diff** as final safety check before commit
5. **In code review**, scrutinize snapshot changes carefully

## Example: Good vs Bad Changes

### Good Change (Accept It)

```diff
- "shares": "60000",        # String
+ "shares": 60000,          # Number (intentional improvement)
```

### Bad Change (Reject It)

```diff
- "shares": 60000,          # Correct value
+ "shares": 0,              # BUG! Parser broke
```

The workflow forces you to see both cases and decide.
