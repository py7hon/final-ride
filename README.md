# Final Ride 🚀

**Final Ride** is a secure, decentralized file management tool built on **Ethereum Swarm**. It allows you to encrypt, chunk, and upload files to the Swarm network, and download them securely.

## Features

- **🛡️ Secure Encryption**: Files are encrypted with AES-256-GCM (Go-compatible across Web & Desktop).
- **🧩 Smart Chunking**: Large files (up to 10MB chunks) are automatically processed with integrity verification.
- **🌐 Web & Desktop**: Purely graphical Windows app, CLI, and a high-performance Web interface.
- **🚀 Auto-Download**: Share direct links (`?download=CID`) that trigger automatic downloads on the web.
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
The project includes a **Swarm Web Downloader/Uploader** for browser-native access.

1. Navigate to `cmd/web/index.html` or host it on your server.
2. **Auto-Download**: Simply visit `index.html?download=CID` to start an automatic download.
3. **Secure Upload**: Drag and drop files to upload with optional AES-256-GCM encryption.
4. **Shareable Links**: Copy the direct "Final Ride" link generated after every upload.

## Configuration

Edit `config.yaml` to customize your experience:

```yaml
swarm_api: https://api.gateway.ethswarm.org
web_url: https://final-ride.ethswarm.org
download_link: "http://localhost:8080/index.html?download=%s"
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
# Via CID
.\final-ride-cli.exe download <Metadata-CID>

# Via Shareable URL (Directly pasted)
.\final-ride-cli.exe download "http://localhost:8080/index.html?download=Qmb..."
```

### GUI (`final-ride-gui.exe`)

1. **Launch**: Double-click `final-ride-gui.exe` (no terminal window will appear).
2. **Branding**: Enjoy the new **Montserrat** powered interface with the "FINAL RIDE" branding.
3. **Upload Tab**:
   - Select your file and toggle encryption.
   - Watch the **Live Progress** and **Transfer Speed**.
   - **Share**: Copy the generated **Shareable Link** to send to others.
4. **Download Tab**:
   - Paste a **Metadata CID** or a full **Shareable URL**.
   - Use the **Paste** button next to the input for quick clipboard access.
   - Files are fetched, integrity-checked, and decrypted automatically.
5. **Settings**: Customize your default download directory and theme instantly.

## Project Structure

- `cmd/cli`: Command-line tool entry point.
- `cmd/gui`: Desktop GUI entry point (Gio UI).
- `internal/finalride`: Shared core logic (Crypto, Swarm, Chunking).
