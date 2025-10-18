# etc_meisai_scraper統合のための仕様書

## 概要

[etc_meisai_scraper](https://github.com/yhonda-ohishi/etc_meisai_scraper)は、ETC明細をWebスクレイピングで自動取得するGoモジュールです。
desktop-serverに統合することで、フロントエンドからETCデータのダウンロード機能を提供できます。

## 現状分析

### etc_meisai_scraperの特徴

**提供するgRPCサービス:**
- `DownloadService` - ETC明細のダウンロード機能
  - `DownloadSync` - 同期ダウンロード
  - `DownloadAsync` - 非同期ダウンロード
  - `GetJobStatus` - ジョブステータス取得
  - `GetAllAccountIDs` - 全アカウントID取得

**protoファイル:**
- `src/proto/download.proto` - DownloadService定義

**依存関係:**
- Playwright (ブラウザ自動化)
- database/sql (MySQL接続)

### 統合可否の検討

#### ✅ 統合可能な点

1. **gRPCサービス提供**: db_serviceと同様にgRPCサービスとして提供されている
2. **明確なAPI**: protoファイルで定義されたAPI
3. **モジュール化**: 独立したGoモジュールとして設計されている

#### ⚠️ 統合上の課題

1. **Playwright依存**: ブラウザ自動化が必要で、desktop-serverのバイナリサイズが大幅に増加する可能性
2. **実行環境**: Playwrightはブラウザバイナリをダウンロード・実行するため、環境依存性が高い
3. **リソース消費**: スクレイピング処理は重く、デスクトップアプリに組み込むと応答性が低下する可能性
4. **protoコード未生成**: リポジトリに生成済みの`.pb.go`ファイルがない

## 推奨アプローチ

### ❌ 推奨しない: 直接統合

以下の理由から、etc_meisai_scraperをdesktop-serverに直接統合することは**推奨しません**:

```
❌ バイナリサイズの肥大化
❌ 環境依存性の増加（Playwright）
❌ パフォーマンスへの悪影響
❌ デスクトップアプリの複雑化
```

### ✅ 推奨: 別プロセスとして実行

etc_meisai_scraperは**別プロセス**として実行し、desktop-serverからgRPCクライアントとして接続する方式を推奨:

```
┌─────────────────────────────────────┐
│ desktop-server.exe                  │
│ - db_service統合（同一プロセス）    │
│ - gRPC-Webプロキシ                  │
│ - フロントエンド提供                │
└─────────────────────────────────────┘
         │
         │ gRPC Client
         ↓
┌─────────────────────────────────────┐
│ etc_meisai_scraper.exe (別プロセス) │
│ - DownloadService提供               │
│ - Playwrightでスクレイピング        │
└─────────────────────────────────────┘
```

## 必要な対応

### 1. etc_meisai_scraperリポジトリ側

#### A. Protoファイルの生成

**ファイル: `.github/workflows/generate-proto.yml`** または既存のMakefile

```yaml
name: Generate Proto Files

on:
  push:
    branches: [main, master]

jobs:
  generate:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4

      - name: Set up Go
        uses: actions/setup-go@v5
        with:
          go-version: '1.21'

      - name: Install protoc
        run: |
          apt-get update && apt-get install -y protobuf-compiler
          go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
          go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest

      - name: Generate proto files
        run: |
          make proto
          # または
          protoc --go_out=. --go-grpc_out=. src/proto/download.proto

      - name: Commit generated files
        run: |
          git config user.name "GitHub Actions"
          git config user.email "actions@github.com"
          git add src/pb/*.go
          git commit -m "Generate proto files" || true
          git push
```

生成されるファイル:
- `src/pb/download.pb.go`
- `src/pb/download_grpc.pb.go`

#### B. Registry パッケージの追加（オプション）

db_serviceと同様のRegistryパターンを実装（ただし、別プロセス推奨のため優先度は低い）

**ファイル: `src/registry/registry.go`**

```go
package registry

import (
	"database/sql"
	"log"

	pb "github.com/yhonda-ohishi/etc_meisai_scraper/src/pb"
	"github.com/yhonda-ohishi/etc_meisai_scraper/src/services"
	"google.golang.org/grpc"
)

// ServiceRegistry holds all etc_meisai_scraper gRPC service implementations
type ServiceRegistry struct {
	DownloadService pb.DownloadServiceServer
}

// NewServiceRegistry creates a new service registry
func NewServiceRegistry(db *sql.DB, logger *log.Logger) *ServiceRegistry {
	return &ServiceRegistry{
		DownloadService: services.NewDownloadServiceGRPC(db, logger),
	}
}

// RegisterAll registers all services to the gRPC server
func (r *ServiceRegistry) RegisterAll(server *grpc.Server) {
	if r.DownloadService != nil {
		pb.RegisterDownloadServiceServer(server, r.DownloadService)
		log.Println("Registered: DownloadService")
	}
}

// Register is a convenience function
func Register(server *grpc.Server, db *sql.DB, logger *log.Logger) *ServiceRegistry {
	registry := NewServiceRegistry(db, logger)
	if registry == nil {
		log.Println("Warning: etc_meisai_scraper not available")
		return nil
	}

	registry.RegisterAll(server)
	return registry
}
```

#### C. スタンドアロンサーバーの改善

**ファイル: `main.go`** を改善して、スタンドアロンサーバーとして実行しやすくする

```go
func main() {
	var (
		grpcPort = flag.String("grpc-port", "50052", "gRPC server port")
		dbDSN    = flag.String("db-dsn", "", "Database DSN")
	)
	flag.Parse()

	logger := log.New(os.Stdout, "[ETC-SCRAPER] ", log.LstdFlags)

	// DB接続
	db, err := sql.Open("mysql", *dbDSN)
	if err != nil {
		logger.Fatalf("Failed to connect to database: %v", err)
	}

	// gRPCサーバー起動
	server := grpc.NewServer(db, logger)
	if err := server.Start(*grpcPort); err != nil {
		logger.Fatalf("Failed to start gRPC server: %v", err)
	}
}
```

### 2. desktop-server側

#### A. プロセス管理と自動起動

**ファイル: `internal/etcscraper/manager.go`**

etc_meisai_scraperを必要な時に自動起動し、不要な時は停止する仕組み:

```go
package etcscraper

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	pb "github.com/yhonda-ohishi/etc_meisai_scraper/src/pb"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// Manager manages etc_meisai_scraper process lifecycle
type Manager struct {
	address    string
	binaryPath string
	process    *exec.Cmd
	client     *Client
	autoStart  bool
}

// NewManager creates a new etc_meisai_scraper manager
func NewManager(address, binaryPath string, autoStart bool) *Manager {
	return &Manager{
		address:    address,
		binaryPath: binaryPath,
		autoStart:  autoStart,
	}
}

// Start starts etc_meisai_scraper process if not running
func (m *Manager) Start() error {
	// Check if process is already running
	if m.process != nil && m.process.ProcessState == nil {
		log.Println("etc_meisai_scraper is already running")
		return nil
	}

	// Find binary path
	if m.binaryPath == "" {
		// Look for binary in same directory as desktop-server
		exePath, err := os.Executable()
		if err != nil {
			return fmt.Errorf("failed to get executable path: %w", err)
		}
		dir := filepath.Dir(exePath)
		m.binaryPath = filepath.Join(dir, "etc_meisai_scraper.exe")
	}

	// Check if binary exists
	if _, err := os.Stat(m.binaryPath); os.IsNotExist(err) {
		return fmt.Errorf("etc_meisai_scraper.exe not found at %s", m.binaryPath)
	}

	// Start process
	log.Printf("Starting etc_meisai_scraper at %s", m.binaryPath)
	m.process = exec.Command(m.binaryPath, "--grpc-port", "50052")
	m.process.Stdout = os.Stdout
	m.process.Stderr = os.Stderr

	if err := m.process.Start(); err != nil {
		return fmt.Errorf("failed to start etc_meisai_scraper: %w", err)
	}

	// Wait for service to be ready
	if err := m.waitForReady(10 * time.Second); err != nil {
		m.Stop()
		return fmt.Errorf("etc_meisai_scraper failed to start: %w", err)
	}

	log.Printf("etc_meisai_scraper started successfully (PID: %d)", m.process.Process.Pid)
	return nil
}

// Stop stops etc_meisai_scraper process
func (m *Manager) Stop() error {
	if m.client != nil {
		m.client.Close()
		m.client = nil
	}

	if m.process != nil && m.process.ProcessState == nil {
		log.Printf("Stopping etc_meisai_scraper (PID: %d)", m.process.Process.Pid)
		if err := m.process.Process.Kill(); err != nil {
			return fmt.Errorf("failed to stop etc_meisai_scraper: %w", err)
		}
		m.process.Wait()
		m.process = nil
	}

	return nil
}

// GetClient returns a gRPC client, starting the process if needed
func (m *Manager) GetClient() (*Client, error) {
	// If client exists and connection is alive, return it
	if m.client != nil {
		return m.client, nil
	}

	// Auto-start if enabled
	if m.autoStart {
		if err := m.Start(); err != nil {
			return nil, fmt.Errorf("failed to auto-start etc_meisai_scraper: %w", err)
		}
	}

	// Create client
	client, err := NewClient(m.address)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to etc_meisai_scraper: %w", err)
	}

	m.client = client
	return client, nil
}

// waitForReady waits for etc_meisai_scraper to be ready
func (m *Manager) waitForReady(timeout time.Duration) error {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return fmt.Errorf("timeout waiting for etc_meisai_scraper to be ready")
		case <-ticker.C:
			conn, err := grpc.DialContext(ctx, m.address,
				grpc.WithTransportCredentials(insecure.NewCredentials()),
				grpc.WithBlock(),
			)
			if err == nil {
				conn.Close()
				return nil
			}
		}
	}
}

// IsRunning checks if etc_meisai_scraper is running
func (m *Manager) IsRunning() bool {
	return m.process != nil && m.process.ProcessState == nil
}
```

#### B. gRPCクライアントの実装

**ファイル: `internal/etcscraper/client.go`**

```go
package etcscraper

import (
	"context"
	"fmt"
	"time"

	pb "github.com/yhonda-ohishi/etc_meisai_scraper/src/pb"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// Client wraps gRPC client for etc_meisai_scraper
type Client struct {
	conn            *grpc.ClientConn
	downloadService pb.DownloadServiceClient
}

// NewClient creates a new client for etc_meisai_scraper
func NewClient(address string) (*Client, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	conn, err := grpc.DialContext(ctx, address,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithBlock(),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to etc_meisai_scraper: %w", err)
	}

	return &Client{
		conn:            conn,
		downloadService: pb.NewDownloadServiceClient(conn),
	}, nil
}

// GetDownloadService returns the Download service client
func (c *Client) GetDownloadService() pb.DownloadServiceClient {
	return c.downloadService
}

// Close closes the gRPC connection
func (c *Client) Close() error {
	if c.conn != nil {
		return c.conn.Close()
	}
	return nil
}

// DownloadAsync starts an async download job
func (c *Client) DownloadAsync(ctx context.Context, accounts []string, fromDate, toDate string) (string, error) {
	req := &pb.DownloadRequest{
		Accounts: accounts,
		FromDate: fromDate,
		ToDate:   toDate,
	}

	resp, err := c.downloadService.DownloadAsync(ctx, req)
	if err != nil {
		return "", err
	}

	return resp.JobId, nil
}

// GetJobStatus retrieves job status
func (c *Client) GetJobStatus(ctx context.Context, jobID string) (*pb.JobStatus, error) {
	req := &pb.GetJobStatusRequest{
		JobId: jobID,
	}

	return c.downloadService.GetJobStatus(ctx, req)
}
```

#### C. main.goでの使用例

**ファイル: `main.go`**

```go
package main

import (
	"desktop-server/internal/etcscraper"
	"log"
)

func main() {
	// ... 既存の初期化コード ...

	// etc_meisai_scraperマネージャーを作成（自動起動有効）
	scraperManager := etcscraper.NewManager("localhost:50052", "", true)
	defer scraperManager.Stop()

	// フロントエンドからリクエストがあった時に自動起動
	// 例: システムトレイメニューやHTTPハンドラーから呼び出し
	go func() {
		// 使用例: ダウンロードジョブを開始
		client, err := scraperManager.GetClient() // 自動起動される
		if err != nil {
			log.Printf("Failed to get scraper client: %v", err)
			return
		}

		// ダウンロード処理を実行
		// ...
	}()

	// ... 既存のサーバー起動コード ...
}
```

#### D. システムトレイメニューへの追加

**ファイル: `systray/tray.go`**

```go
// ETC明細ダウンロード機能をメニューに追加
mETCDownload := systray.AddMenuItem("Download ETC Data", "Download ETC meisai data")
go func() {
	for {
		<-mETCDownload.ClickedCh
		// etc_meisai_scraperを起動してダウンロード
		client, err := scraperManager.GetClient()
		if err != nil {
			log.Printf("Failed to start ETC scraper: %v", err)
			continue
		}

		// 非同期ダウンロードを開始
		jobID, err := client.DownloadAsync(context.Background(), nil, "", "")
		if err != nil {
			log.Printf("Failed to start download: %v", err)
			continue
		}

		log.Printf("ETC download started, job ID: %s", jobID)
	}
}()
```

#### E. gRPC-Webプロキシの設定（オプション）

**ファイル: `server/http.go`** にetc_meisai_scraperへのプロキシ追加

```go
// フロントエンドから直接アクセス可能にする場合
// scraperManagerをHTTPサーバーに渡して、プロキシ設定を追加
```

#### C. README.mdの更新

etc_meisai_scraperの使用方法を追記:

```markdown
### etc_meisai_scraper Integration (Optional)

Desktop Server can integrate with [etc_meisai_scraper](https://github.com/yhonda-ohishi/etc_meisai_scraper) for ETC data download features.

**Separate Process Mode (Recommended):**

1. Run etc_meisai_scraper as a separate process:
   ```bash
   etc_meisai_scraper.exe --grpc-port 50052 --db-dsn "user:password@tcp(localhost:3306)/dbname"
   ```

2. Configure desktop-server to connect:
   ```bash
   ETC_SCRAPER_ADDRESS=localhost:50052
   ```

**Services provided:**
- `DownloadService` - ETC明細ダウンロード
  - DownloadSync - 同期ダウンロード
  - DownloadAsync - 非同期ダウンロード
  - GetJobStatus - ジョブステータス取得
  - GetAllAccountIDs - アカウント一覧取得
```

## メリット・デメリット

### 別プロセス方式（推奨）

**メリット:**
- ✅ desktop-serverのバイナリサイズが小さいまま
- ✅ 環境依存性の分離
- ✅ スクレイピング処理がデスクトップアプリに影響しない
- ✅ etc_meisai_scraperを独立して更新可能
- ✅ 必要な時だけ起動可能

**デメリット:**
- ⚠️ 2つのプロセスを管理する必要がある
- ⚠️ プロセス間通信のオーバーヘッド（ただし軽微）

### 同一プロセス統合方式（非推奨）

**メリット:**
- ✅ 1つのバイナリで完結

**デメリット:**
- ❌ バイナリサイズが大幅に増加
- ❌ Playwright依存による環境依存性
- ❌ スクレイピング処理でUIが重くなる
- ❌ 複雑性の増加

## 実装チェックリスト

### etc_meisai_scraperリポジトリ

- [ ] protoファイルからコード生成（download.pb.go, download_grpc.pb.go）
- [ ] 生成されたファイルをリポジトリにコミット
- [ ] （オプション）Registryパッケージの追加
- [ ] スタンドアロンサーバーとして実行可能にする
- [ ] README.mdにスタンドアロン実行方法を追記
- [ ] リリースでバイナリを配布

### desktop-serverリポジトリ

- [ ] `internal/etcscraper/client.go`を作成
- [ ] gRPC-Webプロキシにetc_meisai_scraperを追加（オプション）
- [ ] 環境変数`ETC_SCRAPER_ADDRESS`の設定を追加
- [ ] README.mdにetc_meisai_scraper統合方法を追記
- [ ] リリースにetc_meisai_scraperのprotoファイルを追加

## 結論

**推奨: 別プロセス方式 + 自動起動管理**

etc_meisai_scraperは、desktop-serverとは別プロセスとして実行し、
gRPCクライアントで接続する方式を推奨します。

### 自動起動の仕組み

```go
// Manager を使用した自動起動
scraperManager := etcscraper.NewManager("localhost:50052", "", true)

// 必要な時に GetClient() を呼ぶだけで自動起動
client, err := scraperManager.GetClient() // etc_meisai_scraper.exe が自動起動される
```

### この方式のメリット

- ✅ **オンデマンド起動**: 必要な時だけプロセスが起動
- ✅ **軽量**: desktop-serverは軽量なまま
- ✅ **透過的**: フロントエンドからは意識せずに使用可能
- ✅ **管理簡単**: Manager が起動・停止を自動管理
- ✅ **リソース効率**: 使わない時はメモリを消費しない

### プロセスライフサイクル

```
1. desktop-server起動
   ↓
2. システムトレイメニューで "Download ETC Data" をクリック
   ↓
3. scraperManager.GetClient() が呼ばれる
   ↓
4. etc_meisai_scraper.exe が自動起動（初回のみ）
   ↓
5. ダウンロード処理を実行
   ↓
6. desktop-server終了時に etc_meisai_scraper も停止
```

### db_serviceとの違い

| 特性 | db_service | etc_meisai_scraper |
|------|-----------|-------------------|
| 統合方式 | 同一プロセス | 別プロセス（自動起動） |
| 理由 | 軽量なDB操作のみ | Playwright依存の重い処理 |
| 起動 | desktop-server起動時 | 必要な時に自動起動 |
| リソース | 常に少量 | 必要な時だけ大量 |

この方式により、desktop-serverは軽量なままで、
ETC明細ダウンロード機能を透過的に提供できます。
