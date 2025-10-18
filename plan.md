# 全体システム仕様書

## システム概要

- **製品名**: YourApp
- **種別**: ローカルデータベース管理ツール（Windows Desktop Application）
- **配布形態**: 単一実行ファイル + タスクトレイ常駐

## アーキテクチャ全体図

```
┌─────────────────────────────────────────────────────┐
│ YourApp.exe (単一バイナリ)                            │
├─────────────────────────────────────────────────────┤
│                                                       │
│  ┌──────────────┐  ┌─────────────────────────────┐ │
│  │タスクトレイUI│  │ HTTPサーバー (localhost:8080)│ │
│  │(systray)    │  ├─────────────────────────────┤ │
│  └──────────────┘  │ gRPC-Webプロキシ            │ │
│                    │ ↓                           │ │
│                    │ gRPCサーバー                │ │
│                    │ ↓                           │ │
│                    │ DB接続層                    │ │
│                    │ ↓                           │ │
│                    │ フロントエンド配信          │ │
│                    │ (埋め込み + GitHub自動更新) │ │
│                    └─────────────────────────────┘ │
│                            ↓                        │
│                    ┌─────────────┐                  │
│                    │ローカルDB    │                  │
│                    │SQL Server   │                  │
│                    │MySQL        │                  │
│                    └─────────────┘                  │
└─────────────────────────────────────────────────────┘
                          ↑
                  ブラウザ (http://localhost:8080)
                  gRPC-Web (Protocol Buffers)
```

---

## リポジトリ構成

### yourapp-backend (メインリポジトリ)

```
backend/
├── proto/
│   └── database.proto          # Protocol Buffers定義
├── generated/go/               # protoc自動生成
│   ├── database.pb.go
│   └── database_grpc.pb.go
├── server/
│   ├── grpc.go                 # gRPCサーバー実装
│   ├── db.go                   # DB接続
│   └── http.go                 # HTTP + gRPC-Web
├── systray/
│   └── tray.go                 # タスクトレイUI
├── updater/
│   └── frontend.go             # フロントエンド自動更新
├── main.go
├── Makefile
├── go.mod
└── .github/workflows/
    └── release.yml
```

**役割:**
- Protocol Buffers定義管理
- gRPCサーバー実装
- DB接続・クエリ実行
- アプリケーションロジック
- 配布バイナリ生成

**配布物 (GitHub Release):**
- `YourApp-Setup.exe` (インストーラー)
- `database.proto` (Frontend用)

### yourapp-frontend
```
frontend/
├── scripts/
│   └── download-proto.sh       # proto自動取得
├── proto/                      # ダウンロード先 (gitignore)
├── generated/ts/               # protoc自動生成 (gitignore)
│   ├── database_pb.ts
│   └── database_grpc_web_pb.ts
├── src/
│   ├── api/
│   │   └── client.ts           # gRPC-Webクライアント
│   ├── components/
│   ├── pages/
│   └── App.tsx
├── package.json
├── vite.config.ts
└── .github/workflows/
    └── build.yml
**役割:**
- UIコンポーネント
- gRPC-Webクライアント実装
- ユーザーインターフェース

**配布物 (GitHub Release):**
- `dist.zip` (ビルド済みフロントエンド)

---

## 技術スタック

| レイヤー | 技術 |
|---------|------|
| Backend | Go 1.21+ |
| gRPC | google.golang.org/grpc |
| gRPC-Web | github.com/improbable-eng/grpc-web |
| DB Driver | go-mssqldb, go-sql-driver/mysql |
| UI (Tray) | github.com/getlantern/systray |
| Frontend | React 18 + TypeScript + Vite |
| gRPC Client | grpc-web |
| Style | Tailwind CSS |
| 型定義 | Protocol Buffers |
| ビルド | Inno Setup (Windows Installer) |

---

## Protocol Buffers定義

**proto/database.proto:**

```protobuf
syntax = "proto3";
package database;

service DatabaseService {
  rpc QueryDatabase(QueryRequest) returns (QueryResponse);
  rpc StreamQuery(StreamQueryRequest) returns (stream QueryRow);
  rpc GetTables(GetTablesRequest) returns (GetTablesResponse);
  rpc ExecuteSQL(ExecuteRequest) returns (ExecuteResponse);
}

message QueryRequest {
  string sql = 1;
  repeated string params = 2;
}

message QueryResponse {
  repeated Row rows = 1;
  int32 count = 2;
}

message Row {
  map<string, string> columns = 1;
}

message StreamQueryRequest {
  string sql = 1;
}

message QueryRow {
  map<string, string> columns = 1;
}

message GetTablesRequest {}

message GetTablesResponse {
  repeated string tables = 1;
}

message ExecuteRequest {
  string sql = 1;
  repeated string params = 2;
}

message ExecuteResponse {
  int32 affected_rows = 1;
}
```

---

## 通信フロー

### 1. アプリ起動

1. YourApp.exe 実行
2. DB接続確認
3. gRPCサーバー起動
4. HTTPサーバー起動 (localhost:8080)
5. フロントエンド更新チェック (バックグラウンド)
6. タスクトレイアイコン表示

### 2. ユーザー操作

1. タスクトレイ右クリック → 「アプリを開く」
2. ブラウザで http://localhost:8080 を開く
3. React UI読み込み
4. gRPC-Webクライアント初期化

### 3. クエリ実行

```
Browser (React)
  ↓ QueryRequest (Protocol Buffers)
gRPC-Web Proxy
  ↓ gRPC
gRPCサーバー
  ↓ SQL
ローカルDB
  ↓ 結果
gRPCサーバー
  ↓ QueryResponse (Protocol Buffers)
gRPC-Web Proxy
  ↓
Browser (完全型安全)
```