# T00: <title>

## Goal
<two sentences: what the code must do and why>

## Implement
- `function_name(args) -> return`: <behaviour>. Edge cases: <...>. Errors: <...>.

## Imitate
<example file and what to copy from it: structure, error handling, naming>

## Data
The gate gives every attempt a fresh, migrated, seeded database in $DATABASE_URL (and
$TEST_DATABASE_URL). Tests must use these variables, never a hard-coded URL.
Test logins: $ADMIN_EMAIL / $ADMIN_PASSWORD and $TEST_USER_EMAIL / $TEST_USER_PASSWORD.

## Do not
- change any signature in the contract files
- add dependencies
- touch files not listed in TASKS.json `files`
