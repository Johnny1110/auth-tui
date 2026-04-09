# Authc User Guide

<br>

---

<br>

## Desc

`authc` is a lightweight, terminal-based Two-Factor Authentication (2FA) manager.
It generates TOTP codes based on your saved secrets and stores them locally in a JSON configuration file.

---

## Installation & Setup

### 1. Prerequisites
- **Go installed**: Ensure you have Go (1.18+) installed on your system.
- **Clipboard support**:
- **Linux**: Requires `xclip` or `xsel` (e.g., `sudo apt install xclip`).
- **macOS/Windows**: Works out of the box.

### 2. Compile from Source
Navigate to your project directory and run:

```bash
go build -o authc main.go
```

<br>
<br>

### 3. Add to System PATH

To run `authc` from any directory, move the binary to a location in your PATH.

For macOS/Linux:

```bash
# Move the binary
sudo mv authc /usr/local/bin/

# Verify installation
authc --help
```

<br>

For Windows:

1. Create a folder (e.g., C:\bin).

2. Move authc.exe into that folder.

3. Add C:\bin to your System Environment Variables under Path.

<br>

---

<br>

## Configuration

The application automatically creates and manages a configuration file at:

`~/.authc-config.json` (Home Directory)

The data is stored in plain-text JSON format:

```json
[
    {
        "name": "GitHub",
        "secret": "JBSWY3DPEHPK3PXP"
    }
]
```

<br>

---

<br>

## How to Use

Launch the app by simply typing:

```bash
authc
```

<br>

### Keybindings & Controls

| Category            | Key            | Action                                                     |
|---------------------|----------------|------------------------------------------------------------|
| Navigation          | ↑ / k          | Move cursor up                                             |
|                     | ↓ / j          | Move cursor down                                           |
| Account Management  | n              | New: Add a new 2FA account                                 |
|                     | r              | Rename: Change the label of the selected account           |
|                     | d              | Delete: Remove the selected account (requires confirmation)|
| Ordering            | K (Shift+k)    | Move Up: Swap current account with the one above           |
|                     | J (Shift+j)    | Move Down: Swap current account with the one below         |
| General             | c              | Copy: Copy the current 6-digit TOTP code to clipboard      |
|                     | q / Ctrl+C     | Quit: Exit the application                                 |
|                     | Esc            | Cancel: Exit the current input mode or deletion prompt     |

<br>

---

<br>

## Tips

* Secret Format: When adding a new account, the secret should be the Base32 string provided by the service (e.g., Google, GitHub). Spaces are automatically removed.
* Security: Since the secret keys are stored in your home directory in plain text, ensure your machine uses disk encryption (like FileVault or BitLocker) and that you keep your laptop locked.
* Syncing: You can backup or sync your ~/.authc-config.json across machines to keep your 2FA codes consistent.