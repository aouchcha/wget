🚀 Go-Wget — A Lightweight Wget Clone in Go

Go-Wget is a command-line download utility written in pure Go.
It supports single-file downloads, rate limiting, background mode, output customization, and basic website mirroring—similar to GNU wget.

✨ Features
📄 Single File Download

Save file with a specific name (-O)

Save into a specific directory (-P)

Automatic filename extraction from URL

⚡ Download Controls

Background mode (-B) — logs output to wget-log

Rate limiting (--rate-limit=200k, 500k, 2m, etc.)

🌍 Mirroring Mode

(--mirror)

Downloads the main page and prepares the structure for recursive mirroring

Reject file types (-R=pdf,zip,exe)

Exclude directories (-X=/admin,/private)

Convert links for offline usage (--convert-links)

Works together with rate limit & background mode

🧹 Safety & Validation

Validates conflicting flags (e.g., cannot use -O with --mirror)

Ensures proper directory creation

URL validation & parsing

📦 Installation
git clone https://github.com/yourusername/go-wget
cd go-wget
go build -o go-wget

🛠️ Usage
Basic Syntax
go-wget [flags] <URL>

🔧 Available Flags
Flag	Description
-O=<file>	Save output as a specific filename
-P=<path>	Save file inside a directory
-B	Run in background mode (write logs to wget-log)
--rate-limit=<speed>	Limit download speed (supports k, kb, m, mb)
--mirror	Enable mirror mode
--convert-links	Rewrite links for offline viewing
-R=<types>	Reject certain file extensions (pdf,zip,exe)
-X=<dirs>	Exclude directories (/admin,/private)
🚀 Examples
1️⃣ Download a Single File
go-wget https://example.com/file.zip

2️⃣ Save With a Custom Filename & Directory
go-wget -O=myfile.bin -P=./downloads https://example.com/file.bin

3️⃣ Rate-Limited Download
go-wget --rate-limit=200k https://example.com/large.iso

4️⃣ Background Download (writes to wget-log)
go-wget -B https://example.com/file.zip

🌐 Mirror a Website
go-wget \
  --mirror \
  --convert-links \
  -R=pdf,zip,exe \
  -X=/admin,/private \
  --rate-limit=500k \
  -P=./mirror_site \
  -B \
  https://example.com


This will:

✔ Create a folder mirror_site/example.com/
✔ Download the main HTML page
✔ Apply rate limiting
✔ Reject unwanted file types
✔ Exclude restricted folders
✔ Convert links for offline navigation
✔ Log all events to wget-log

🏗 Project Structure

Your project includes:

DownloadConfig — holds parsed flags

ParseArgs() — handles CLI parsing & validation

ExecuteDownload() — chooses single or mirror mode

downloadFile() — performs the actual download

downloadWithRateLimit() — controlled-speed download pipeline

mirrorWebsite() — foundations for recursive mirroring

Additional helpers for logging, filename parsing, rate converting...

🧪 Built-in Examples (from main.go)

Your main() function includes two ready-to-run test scenarios:

Single download with rate limit

Full mirror mode test

You can run them with:

go run .

🧭 Roadmap

Full recursive mirroring

HTML parsing for assets (CSS, images, scripts)

Parallel downloads

Retry logic & resume support

HTTPS certificate configuration

❤️ Contributing

PRs are welcome! Feel free to open issues or suggest features.

📜 License

MIT License (modify if needed)
