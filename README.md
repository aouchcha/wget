# wget (Go Implementation)

A command-line utility written in Go for downloading files and recursively mirroring websites, inspired by GNU Wget. Supports single and batch downloads, rate limiting, background mode, and advanced mirroring options.

## Features

- Download single files from HTTP/HTTPS URLs
- Batch download from a list of URLs in a file
- Recursive website mirroring (`--mirror`)
- Exclude file types (`-R`, `--reject`) and folders (`-X`, `--exclude`) during mirroring
- Output to a specific file or directory (`-O`, `-P`)
- Limit download speed (`--limit-rate`)
- Background mode with logging (`-B`)
- Convert links for offline browsing (`--convert-links`)
- Progress bar and human-readable output

## Usage

```sh
go run . [OPTIONS] URL
```

### Examples

- Download a single file:
  ```sh
  go run . https://example.com/file.zip
  ```

- Download to a specific file:
  ```sh
  go run . -O=output.zip https://example.com/file.zip
  ```

- Batch download from a file:
  ```sh
  go run . -i urls.txt
  ```

- Mirror a website recursively:
  ```sh
  go run . --mirror https://example.com/
  ```

- Mirror with rejected file types and excluded folders:
  ```sh
  go run . --mirror -R=jpg,png -X=/private/,/tmp/ https://example.com/
  ```

- Limit download speed to 500KB/s:
  ```sh
  go run . --limit-rate=500k https://example.com/file.zip
  ```

- Run in background mode (output to `wget-log`):
  ```sh
  go run . -B https://example.com/file.zip
  ```

## Options

| Option                | Description                                              |
|-----------------------|---------------------------------------------------------|
| `-O FILE`             | Write output to `FILE`                                  |
| `-P DIR`              | Save files to directory `DIR`                           |
| `-i FILE`             | Download URLs listed in `FILE`                          |
| `--rate-limit=RATE`   | Limit download speed (e.g., `500k`, `2m`)               |
| `--mirror`            | Enable recursive mirroring                              |
| `-R`, `--reject=LIST` | Reject files with given extensions (comma-separated)    |
| `-X`, `--exclude=LIST`| Exclude folders (comma-separated, e.g., `/admin/`)      |
| `--convert-links`     | Convert links for offline viewing                       |
| `-B`                  | Run in background mode, log to `wget-log`               |

## Building

Requires Go 1.23 or later.

```sh
go build -o wget
```

