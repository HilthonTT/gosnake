# Contributing to gosnake

Thanks for your interest in improving gosnake. Bug reports, ideas and pull requests are all welcome.

## Project layout

| Path | What lives there |
| --- | --- |
| `cmd/gosnake/` | CLI entry point |
| `pkg/snake/` | Game engine: pure Go, no I/O or UI |
| `internal/tui/` | Bubble Tea views, theme and components |
| `internal/server/` | SSH multiplayer server |
| `internal/data/` | SQLite leaderboard storage |
| `internal/tipsapi/` | Client for the online leaderboard |
| `tips-api/` | ASP.NET Core 11 service and its tests |
| `docs/assets/` | Logo, social preview and README GIFs |
| `scripts/demo/` | Records the README GIFs: `go run ./scripts/demo -out docs/assets` |

## Development

You need Go (see `go.mod` for the version). The .NET 11 SDK is only needed if you work on `tips-api/`.

```sh
make test       # go test -race ./...
make lint       # golangci-lint run
make api-test   # dotnet test tips-api/Tips.slnx
make run        # play from source
```

## Pull requests

- Keep each PR focused on one change, and add or update tests for behaviour changes.
- Run `make test` and `make lint` before you push. CI runs the same checks on Linux, macOS and Windows.
- Use [Conventional Commits](https://www.conventionalcommits.org/) for commit messages, for example `fix(tui): ...` or `feat(server): ...`. The release changelog is built from them.
