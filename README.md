# Desktop Server

Windows Desktop Application for local database management with gRPC-Web API.

## Features

- **Single Binary**: Distributes as a single executable file
- **System Tray**: Runs in the system tray for easy access
- **gRPC-Web API**: Modern API using Protocol Buffers
- **Multi-Database Support**: SQL Server and MySQL
- **Web UI**: Browser-based interface (React + TypeScript)

## Architecture

```
┌─────────────────────────────────────────────────────┐
│ desktop-server.exe (Single Binary)                  │
├─────────────────────────────────────────────────────┤
│                                                       │
│  ┌──────────────┐  ┌─────────────────────────────┐ │
│  │System Tray UI│  │ HTTP Server (localhost:8080)│ │
│  │(systray)     │  ├─────────────────────────────┤ │
│  └──────────────┘  │ gRPC-Web Proxy              │ │
│                    │ ↓                           │ │
│                    │ gRPC Server                 │ │
│                    │ ↓                           │ │
│                    │ DB Connection Layer         │ │
│                    └─────────────────────────────┘ │
│                            ↓                        │
│                    ┌─────────────┐                  │
│                    │Local DB     │                  │
│                    │SQL Server   │                  │
│                    │MySQL        │                  │
│                    └─────────────┘                  │
└─────────────────────────────────────────────────────┘
                          ↑
                  Browser (http://localhost:8080)
                  gRPC-Web (Protocol Buffers)
```

## Prerequisites

### For Users

- Windows 7 or later
- Database (SQL Server or MySQL)

**No installation required!** Just download and run `desktop-server.exe`

### For Developers

- Go 1.21 or later
- Protocol Buffers compiler (protoc)
- Database (SQL Server or MySQL)

### Install Development Tools

```bash
# Protocol Buffers tools
make install-tools
```

## Download & Run

1. Download `desktop-server.exe` from [GitHub Releases](https://github.com/yourusername/desktop-server/releases)
2. Double-click to run
3. The app will appear in your system tray
4. Right-click the tray icon and select "Open App"

## Building from Source

```bash
# Build the application (with console window for debugging)
make build

# Build for GUI mode (no console window)
make build-gui

# Run the application
make run

# Update frontend to latest version
make update-frontend
```

## Release Process

This project uses GitHub Actions for automated releases.

### Creating a New Release

1. Update version in code if needed
2. Commit all changes
3. Create and push a new tag:

```bash
git tag v1.0.0
git push origin v1.0.0
```

4. GitHub Actions will automatically:
   - Build the application
   - Download latest frontend
   - Create a GitHub Release
   - Upload `desktop-server.exe` as a release asset

### Manual Release

```bash
# Download latest frontend
go run . -update

# Build for release
make build-gui

# Or directly with go
go build -ldflags="-H windowsgui" -o desktop-server.exe .
```

## Configuration

### Desktop-Server Database (Optional)

Set environment variables for desktop-server's database connection:

#### SQL Server

```bash
DB_DRIVER=sqlserver
DB_SERVER=localhost
DB_PORT=1433
DB_USER=sa
DB_PASSWORD=yourpassword
DB_NAME=master
```

#### MySQL

```bash
DB_DRIVER=mysql
DB_HOST=localhost
DB_PORT=3306
DB_USER=root
DB_PASSWORD=yourpassword
DB_NAME=mysql
```

### db_service Integration (Optional)

Desktop Server integrates with [db_service](https://github.com/yhonda-ohishi/db_service) for additional database services.

To enable db_service features, create a `.env` file in the same directory as `desktop-server.exe`:

```bash
# Required
DB_USER=your_mysql_username
DB_PASSWORD=your_mysql_password

# Optional (defaults shown)
DB_HOST=localhost
DB_PORT=3306
DB_NAME=ryohi_sub_cal
```

Or copy `.env.example` to `.env` and edit:

```bash
cp .env.example .env
# Edit .env with your database credentials
```

**db_service provides these gRPC services:**
- `ETCMeisaiService` - ETC明細管理
- `DTakoUriageKeihiService` - 経費精算管理
- `DTakoFerryRowsService` - フェリー運行管理
- `ETCMeisaiMappingService` - ETC明細マッピング

If `.env` is not configured, desktop-server will run without db_service features (warning messages will appear in logs).

## Running

1. Set database environment variables (optional, see Configuration above)
2. Run the executable: `desktop-server.exe`
3. The app will start in the system tray (no console window)
4. Right-click the tray icon and select:
   - **Open App**: Opens the web interface in your browser
   - **Check for Updates**: Checks for new backend versions on GitHub
   - **Update Frontend**: Downloads latest frontend from releases
   - **About**: Shows version information
   - **Quit**: Exits the application

## Auto-Update Features

Desktop Server includes built-in auto-update functionality for both backend and frontend:

### Backend Updates

- Click "Check for Updates" in the system tray menu
- The app will check GitHub Releases for new backend versions
- If available, it will download and apply the update automatically
- The app will restart with the new version

### Frontend Updates

- Click "Update Frontend" in the system tray menu
- Downloads the latest frontend from the frontend repository
- Restart the application to apply changes
- Or use command line: `desktop-server.exe -update`

### Automatic Frontend Download

- On first run, if frontend is missing, it will be downloaded automatically
- No manual setup required

## Project Structure

```
desktop-server/
├── main.go                 # Application entry point
├── proto/
│   ├── database.proto      # Protocol Buffers definition
│   ├── database.pb.go      # Generated Go code
│   └── database_grpc.pb.go # Generated gRPC code
├── server/
│   ├── db.go               # Database connection layer
│   ├── grpc.go             # gRPC server implementation
│   └── http.go             # HTTP + gRPC-Web proxy
├── systray/
│   └── tray.go             # System tray UI with auto-update
├── updater/
│   └── github.go           # GitHub Release auto-updater
├── desktop-sv/             # Frontend (React + TypeScript)
│   ├── src/
│   ├── package.json
│   └── vite.config.ts
├── Makefile
└── go.mod
```

## API Endpoints

### gRPC Services

- `QueryDatabase`: Execute SQL queries
- `StreamQuery`: Stream query results
- `GetTables`: Get list of database tables
- `ExecuteSQL`: Execute SQL commands (INSERT, UPDATE, DELETE)

### HTTP Endpoints

- `http://localhost:8080/`: Web UI
- `http://localhost:8080/api/`: gRPC-Web API endpoint

## Development

### Generate Proto Files

```bash
make proto
```

### Build Frontend

```bash
cd desktop-sv
npm install
npm run build
```

### Clean Build Artifacts

```bash
make clean
```

## License

MIT
