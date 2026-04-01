# schoologyCLI

Go CLI for the Schoology API.

## Credentials

Get the consumer key and consumer secret from:

```text
https://<schooldomain>.schoology.com/api
```

Export them:

```fish
set -x SCHOOLOGY_KEY your-consumer-key
set -x SCHOOLOGY_SECRET your-consumer-secret
```

Optional:

```fish
set -x SCHOOLOGY_API_BASE https://api.schoology.com/v1
```

## Build

```fish
cd ~/projects/schoologyCLI
go build
```

## Run

```fish
cd ~/projects/schoologyCLI
go run . me
go run . sections
go run . assignments --section 7916825515
go run . upcoming --days 7
```

## Binary

```fish
./schoologyCLI me
./schoologyCLI sections
./schoologyCLI assignments --section 7916825515
./schoologyCLI upcoming --days 7
./schoologyCLI upcoming --days 7 --json
```

## Notes

- `upcoming` uses the section `events` endpoint and filters to assignment-type events.
- `assignments` uses the section `assignments` endpoint.
