# Final Ride 🚀

**Final Ride** is a secure, decentralized file management tool built on **Ethereum Swarm**. It allows you to encrypt, chunk, and upload files to the Swarm network, and download them securely.

## Features

- **🛡️ Secure Encryption**: Files are encrypted with AES-256-GCM (Go-compatible across Web & Desktop).
- **🧩 Smart Chunking**: Large files (up to 10MB chunks) are automatically processed with integrity verification.
- **🌐 Web & Desktop**: Purely graphical Windows app, CLI for power users, and a new Web interface.
- **🏎️ Real-time Feedback**: Live progress bars and transfer speed indicators on all platforms.
- **🎨 Premium UI**: Modern Montserrat typography with immediate theme switching and zero-freeze performance.
- **☁️ Swarm Powered**: Decentralized storage via Ethereum Swarm gateway.

## Installation

Run the following commands to build the project:

```bash
# Build CLI tool
go build -o final-ride-cli.exe ./cmd/cli

# Build GUI (Modern Desktop App - Hidden Console)
go build -o final-ride-gui.exe -ldflags="-H windowsgui" ./cmd/gui
```

## Quick Start: Web Interface 🌐
The project now includes a **Swarm Web Downloader/Uploader**. 

1. Navigate to `cmd/web/index.html`.
2. Open it in any modern browser.
3. Start uploading or downloading (CID-only) directly from your browser with AES-256-GCM security.

## Configuration

Edit `config.yaml` to customize your experience:

```yaml
swarm_api: https://api.gateway.ethswarm.org/bzz
download_link: "https://download.example.com?cid=%s&key=%s"
chunk_size_mb: 10
theme: "dark"           # "light" or "dark"
download_dir: "C:/Downloads"
encrypt_default: true   # Initial state of encryption toggle
```

## Usage

### CLI (`final-ride-cli.exe`)

**Upload a file:**
```bash
# Encrypted (Default)
.\final-ride-cli.exe upload MySecretFile.zip

# Unencrypted
.\final-ride-cli.exe upload PublicImage.png --no-encrypt
```

**Download a file:**
```bash
.\final-ride-cli.exe download <Metadata-CID>
```

### GUI (`final-ride-gui.exe`)

1. **Launch**: Double-click `final-ride-gui.exe` (no terminal window will appear).
2. **Branding**: Enjoy the new **Montserrat** powered interface with the "FINAL RIDE" branding.
3. **Upload Tab**:
   - Select your file via "Browse".
   - Toggle encryption (honors `encrypt_default` initially).
   - Watch the **Live Progress** and **Transfer Speed**.
4. **Download Tab**:
   - Paste the **Metadata CID**.
   - Your file is fetched, integrity-checked, and decrypted automatically.
5. **Settings**: Customize your default download directory and theme instantly.

## Project Structure

- `cmd/cli`: Command-line tool entry point.
- `cmd/gui`: Desktop GUI entry point (Gio UI).
- `internal/finalride`: Shared core logic (Crypto, Swarm, Chunking).
