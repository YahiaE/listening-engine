# Listening Engine

Listening Engine tracks Spotify listening telemetry from Windows and stores it in PostgreSQL.
The repository contains two applications:

- `smtc-watcher/`: a Windows .NET console app that reads Spotify media metadata.
- `cmd/`, `internal/`: a Go HTTP server that authenticates devices, verifies users, and stores listening events.

## Workflow

```text
Spotify -> Windows SMTC -> smtc-watcher -> Go HTTP server -> PostgreSQL
                                      |
                                      `-> Windows Credential Manager
```

1. The watcher starts and looks for credentials in the Windows Credential Manager.
2. Without credentials, it collects an email and requests a six-digit OTP.
3. The Go server sends the OTP through Gmail SMTP and retains it in an LRU cache.
4. After successful verification, the server creates or finds a user and issues a new authentication token.
5. The watcher stores the user ID and raw token locally and sends them with later telemetry.
6. The server hashes the supplied token, authenticates it through its cache or PostgreSQL, and decodes the song.
7. The database de-duplicates songs and assigns each event to a listening session.
8. An event starts a new session after more than 30 minutes since the user's latest event.
9. If authentication fails with a mismatch, the watcher pauses telemetry and starts verification again.

## HTTP endpoints

| Endpoint | Method | Purpose |
| --- | --- | --- |
| `/send-otp` | `POST` | Generate and email an OTP using the `Email` header |
| `/verify-otp` | `POST` | Verify an email/OTP pair and return `user_id` and `token` |
| `/` | `POST` | Accept authenticated song telemetry |

Telemetry requests use these headers:

```text
Auth-Token: <raw-device-token>
User-ID: <user-uuid>
```

The JSON body contains:

```json
{
  "title": "Song title",
  "artist": "Artist name",
  "album": "Album name"
}
```

## Data model

- `users`: registered email addresses and UUID identities.
- `auth_token`: SHA-256-hashed device tokens linked to users.
- `song`: unique, normalized title/artist/album combinations.
- `session`: listening activity periods for a user.
- `event`: a user's timestamped occurrence of a song in a session.

The PostgreSQL function `create_event_and_session` creates or resumes sessions based on the 30-minute gap rule.

## Configuration

Create a `.env` file in the repository root:

```dotenv
DATABASE_URL=postgresql://...
PORT=5000
EMAIL=sender@example.com
PASS=mail-app-password
```

The server loads `.env` through `internal/config/config.go`. Do not commit real database credentials or mail passwords.
If credentials have already been exposed, rotate the database password and Gmail app password.

The Windows watcher uses the server address configured by `ServerUrl` in `smtc-watcher/Program.cs`.

## Run the Go server

From the repository root:

```bash
go run ./cmd/main
```

The server creates missing tables and database functions at startup.

## Run the Windows watcher

Open `smtc-watcher/listening-engine.slnx` in Visual Studio or run:

```powershell
dotnet run --project .\smtc-watcher\smtc-watcher.csproj
```

The watcher must run on Windows 10/11 and has the Spotify application open for SMTC events.
Press Enter in the console to exit.

## Package roles

- `cmd/main/main.go`: application bootstrap and dependency wiring.
- `internal/config`: environment loading.
- `internal/db`: PostgreSQL connection, schema, and session SQL function.
- `internal/server`: routes and authentication handlers.
- `internal/store`: database writes for users, tokens, songs, and events.
- `internal/auth`: token and OTP generation plus token hashing.
- `internal/email`: Gmail SMTP delivery.
- `internal/models`: HTTP and database payload structures.
- `smtc-watcher/Program.cs`: Windows session discovery and telemetry sending.
