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
```

## Binary

```fish
./schoologyCLI me
./schoologyCLI sections
./schoologyCLI assignments --section <section-id>
./schoologyCLI upcoming --days 7
./schoologyCLI upcoming --days 7 --json
```

## Notes

- `upcoming` uses the section `events` endpoint and filters to assignment-type events.
- `assignments` uses the section `assignments` endpoint.
