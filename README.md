# 🚀 Go-Wget — A Lightweight Wget Clone in Go

Go-Wget is a command-line download utility written in pure Go.  
It supports single-file downloads, rate limiting, background mode, output customization, and basic website mirroring—similar to GNU `wget`.

---

## ✨ Features

### 📄 Single File Download
- Save file with a specific name (`-O`)
- Save into a specific directory (`-P`)
- Automatic filename extraction from URL

### ⚡ Download Controls
- Background mode (`-B`) — logs output to `wget-log`
- Rate limiting (`--rate-limit=200k`, `500k`, `2m`, etc.)

### 🌍 Mirroring Mode
(`--mirror`)
- Downloads the main page and prepares the structure for recursive mirroring
- Reject file types (`-R=pdf,zip,exe`)
- Exclude directories (`-X=/admin,/private`)
- Convert links for offline usage (`--convert-links`)
- Works together with rate limit & background mode

### 🧹 Safety & Validation
- Validates conflicting flags (e.g., cannot use `-O` with `--mirror`)
- Ensures proper directory creation
- URL validation & parsing

---

## 📦 Installation

```bash
git clone https://github.com/aouchcha/wget
cd wget
go build -o go-wget
```
---

## 🛠️ Usage
Basic Syntax
```bash
go-wget [flags] <URL>
```
---

## 🔧 Available Flags
Flag	Description: 
-O=<file>	Save output as a specific filename
-P=<path>	Save file inside a directory
-B	Run in background mode (write logs to wget-log)
--rate-limit=<speed>	Limit download speed (supports k, kb, m, mb)
--mirror	Enable mirror mode
--convert-links	Rewrite links for offline viewing
-R=<types>	Reject certain file extensions (pdf,zip,exe)
-X=<dirs>	Exclude directories (/admin,/private)

---

## 🚀 Examples
1️⃣ Download a Single File
```bash
go-wget https://example.com/file.zip
```

2️⃣ Save With a Custom Filename & Directory
```bash
go-wget -O=myfile.bin -P=./downloads https://example.com/file.bin
```
3️⃣ Rate-Limited Download
```bash
go-wget --rate-limit=200k https://example.com/large.iso
  ```

4️⃣ Background Download (writes to wget-log)
```bash
go-wget -B https://example.com/file.zip
```
🌐 Mirror a Website
``` bash
go-wget --mirror -R=jpg,gif https://example.com

go-wget --mirror -X=/assets,/css https://example.com

go-wget --mirror --convert-links https://example.com
```

---

## ❤️ Contributing

PRs are welcome! Feel free to open issues or suggest features.
https://github.com/2001basta
https://github.com/aouchcha
https://github.com/ABouziani
https://github.com/x3alone
---

## contributors

