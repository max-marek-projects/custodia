# Custodia – Secure Password Manager

**Custodia** is a client‑server password manager built with Go. It allows users to securely store and manage credentials, text notes, binary data, and bank card details with versioning and end‑to‑end encryption.

---

## ✨ Features

### Server

- **User registration & authentication** – JWT‑based with refresh tokens.
- **Secret storage** – store any data (login/password, text, binary, card) with metadata.
- **Versioning** – every update creates a new version; rollback to any previous version.
- **Data synchronization** – clients retrieve their latest secrets from the server.
- **gRPC API** with optional TLS.

### Client (CLI)

- **Interactive REPL** – a shell‑like interface for all commands.
- **Authentication** – login, register, refresh access token, logout (single or all devices).
- **Secret management** – create, get, update, delete, rollback, and update metadata for:
  - Credentials (login/password)
  - Arbitrary text
  - Arbitrary binary data
  - Bank card details (with Luhn validation)
- **Secure input** – passwords and secrets are hidden during typing.
- **Clipboard integration** – retrieved secrets are copied to the clipboard with a configurable TTL.
- **Cross‑platform** – works on Windows, Linux, and macOS.

### Security

- All sensitive data is encrypted **on the client** before transmission (AES‑GCM with PBKDF2).
- Authentication uses **JWT** with short‑lived access tokens and long‑lived refresh tokens.
- TLS support for gRPC communication (optional).
- Tokens are stored locally with file permissions `0600`.

---

## 📦 Installation

### From source

```bash
git clone https://github.com/max-marek-projects/custodia.git
cd custodia
```

## 🚀 Usage

### Server usage

Create `.env` file [example](./.env.example) and run

```bash
make server
```

Flags can be set via environment variables or a config file (see `--help`).

### Client usage

```bash
# Open interactive REPL
make client
custodia
custodia> login -l user
custodia> create credentials -n mylogin -l user
custodia> get credentials -n mylogin
custodia> logout
```

**Available commands:**

- `register` – create a new account
- `login` – authenticate and start a session
- `refresh` – renew the access token
- `logout` – end session (optionally on all devices)
- `create` – add a new secret (credentials, text, binary, card‑data)
- `get` – retrieve a secret (copied to clipboard)
- `update` – modify data or metadata of a secret
- `delete` – remove a secret (all versions)
- `rollback` – revert to the previous version

---

## 📁 Project Structure

```text
├── cmd/
│   ├── custodia/     – CLI client
│   └── server/       – gRPC server
├── internal/
│   ├── auth/         – JWT and refresh token handling
│   ├── client/       – client logic, REPL, session, token storage
│   ├── config/       – configuration loading (env, flags, JSON)
│   ├── handlers/     – gRPC handlers
│   ├── interceptors/ – gRPC interceptors (auth, logging)
│   ├── logger/       – slog wrapper with custom levels
│   ├── models/       – data structures (secrets, card, credentials)
│   ├── repository/   – PostgreSQL storage with migrations
│   ├── requests/     – context and metadata helpers
│   ├── server/       – gRPC server setup
│   ├── service/      – business logic (auth, secrets)
│   └── utils/        – encryption, clipboard, formatting, validation
├── pkg/
│   └── proto/        – gRPC protocol definitions
└── migrations/       – database schema migrations
```

---

## 🧪 Testing & Coverage

- **Unit tests** cover >70% of the codebase.
- **Integration tests** for database and gRPC.
- Run with:

```bash
make test
```

---

## 📝 Documentation

All exported functions, types, and packages are documented in the code according to Go standards.  
Run `go doc` to explore.

---

## 🔧 What Could Be Improved (Optional Features from Specification)

The following optional features from the specification are **not yet implemented** but could be added in future releases:

- **OTP (one‑time password)** support – for time‑based tokens (TOTP).
- **Swagger/OpenAPI** documentation for the gRPC API.
- **Build version and date** – display when running `custodia version` (currently only in code, not exposed).
- **Full TUI** – a more interactive terminal interface beyond the current REPL.
- **Push synchronization** – real‑time updates when data changes (currently client‑pull only).

These enhancements would further align the project with the original specification and improve user experience.

---

## 📄 License

[Choose your license, e.g., MIT]

---

## 👤 Author

Max Marek – [GitHub](https://github.com/max-marek-projects)

---

*Built with Go, gRPC, PostgreSQL, and love.*
