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
```

## Configuration

Set environment variables for database connection:

### SQL Server

```bash
DB_DRIVER=sqlserver
DB_SERVER=localhost
DB_PORT=1433
DB_USER=sa
DB_PASSWORD=yourpassword
DB_NAME=master
```

### MySQL

```bash
DB_DRIVER=mysql
DB_HOST=localhost
DB_PORT=3306
DB_USER=root
DB_PASSWORD=yourpassword
DB_NAME=mysql
```

## Running

1. Set database environment variables (optional, see Configuration above)
2. Run the executable: `desktop-server.exe`
3. The app will start in the system tray
4. Right-click the tray icon and select:
   - **Open App**: Opens the web interface in your browser
   - **Check for Updates**: Checks for new versions on GitHub
   - **About**: Shows version information
   - **Quit**: Exits the application

## Auto-Update Feature

Desktop Server includes built-in auto-update functionality:

- Click "Check for Updates" in the system tray menu
- The app will check GitHub Releases for new versions
- If available, it will download and apply the update automatically
- The app will restart with the new version

**For Developers**: Update the version in `updater/github.go` and create a GitHub Release with the new executable.

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
