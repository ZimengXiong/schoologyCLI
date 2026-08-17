# schoologyCLI

Go CLI for the Schoology API.

## Credentials

Get the consumer key and consumer secret from:

```text
https://<schooldomain>.schoology.com/api
```

Persistent shell variables:

```fish
set -x SCHOOLOGY_KEY your-consumer-key
set -x SCHOOLOGY_SECRET your-consumer-secret
```

Inline for one command:

```fish
env SCHOOLOGY_KEY=your-consumer-key SCHOOLOGY_SECRET=your-consumer-secret go run . me
```

On macOS, credentials can instead be stored in Keychain so shells and agents do
not need to carry secrets in their environment:

```sh
security add-generic-password -U -s schoology-cli -a consumer-key -w 'your-consumer-key'
security add-generic-password -U -s schoology-cli -a consumer-secret -w 'your-consumer-secret'
```

Environment variables take precedence over Keychain entries. The CLI never
prints either credential.

Optional:

```fish
set -x SCHOOLOGY_API_BASE https://api.schoology.com/v1
```

## Clone

```fish
git clone https://github.com/ZimengXiong/schoologyCLI.git
cd schoologyCLI
```

## Build

```fish
go build
```

## Run

```fish
go run . me
go run . sections
go run . assignments --section <section-id>
go run . upcoming --days 7
go run . submissions --section <section-id> --assignment <assignment-id>
```

## Binary

```fish
./schoologyCLI me
./schoologyCLI sections
./schoologyCLI sections --course physics --json
./schoologyCLI assignments --section <section-id>
./schoologyCLI assignment --section <section-id> --assignment <assignment-id> --json
./schoologyCLI upcoming --days 7
./schoologyCLI upcoming --days 7 --json
./schoologyCLI submissions --section <section-id> --assignment <assignment-id> --json
./schoologyCLI updates --section <section-id> --limit 10 --json
./schoologyCLI documents --section <section-id> --json
./schoologyCLI pages --section <section-id> --json
./schoologyCLI audit --days 14 --json
./schoologyCLI todos --days 7 --course biochemistry --json
```

## Agent workflow

`audit` is the high-level research command. It scans active sections, fetches
assignments in the due-date window, and checks the signed-in student's
submission revisions. Its stable JSON includes course and assignment IDs,
description, due date, URL, submission evidence, uncertainty, and an
`actionable` flag. `todos` returns only actionable audit items.

Submission states are:

- `submitted`: at least one non-draft Schoology revision is verified.
- `draft`: revisions exist, but all are drafts.
- `not_submitted`: the API returned no revisions.
- `external`: the assignment explicitly references Turnitin and Schoology does
  not verify a submission.
- `unknown`: Schoology's submission endpoint failed; `submission_error`
  preserves the reason instead of guessing.

Use `--include-overdue=false` to restrict an audit to now through the cutoff.
Course filters are case-insensitive substrings across course title, code, and
section title.

## Notes

- `upcoming` uses the section `events` endpoint and filters to assignment-type events.
- `assignments` uses the section `assignments` endpoint. Its `completed` field
  is assignment metadata and should not be treated as proof of a student
  submission.
- `submissions` uses the student-specific submissions endpoint and reports
  `submitted=true` only when at least one non-draft revision exists.
- `assignment` requests attachments and tags in addition to assignment details.
- `updates`, `documents`, and `pages` expose course research surfaces that are
  otherwise easy for an agent to miss.
- All commands are read-only. The CLI does not submit assignments, post updates,
  modify materials, or alter completion state.
- Turnitin submission status is outside Schoology's API and remains an explicit
  external verification gap.
