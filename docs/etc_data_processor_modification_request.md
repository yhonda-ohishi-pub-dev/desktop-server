# etc_data_processor 修正依頼

## 概要
etc_data_processorをdb_serviceと連携させるため、DBClient インターフェースをdb_serviceのgRPC APIに合わせて修正する。

## 現在の問題

### 1. DBClientインターフェースが汎用的すぎる
```go
type DBClient interface {
    SaveETCData(data interface{}) error
}
```
- `interface{}`型では型安全性がない
- db_serviceのproto定義と直接マッピングできない

### 2. db_serviceとの型不一致
**db_service側 (proto定義):**

場所: `C:\go\db_service\src\proto\ryohi.proto`
または: `https://github.com/yhonda-ohishi/db_service/src/proto/ryohi.proto`

```protobuf
message ETCMeisai {
  int64 id = 1;
  optional string date_fr = 2;
  string date_to = 3;              // 必須 - RFC3339形式推奨 (例: 2025-10-18T00:00:00Z)
  string date_to_date = 4;         // 必須 - YYYY-MM-DD形式 (例: 2025-10-18)
  optional string ic_fr = 5;       // 入口IC - 実データの22.3%が空なのでoptional
  string ic_to = 6;                // 必須 - 出口IC
  optional int32 price_bf = 7;
  optional int32 descount = 8;
  int32 price = 9;                 // 必須 - ETC料金
  int32 shashu = 10;               // 必須 - 車種
  optional int32 car_id_num = 11;
  string etc_num = 12;             // 必須 - ETCカード番号
  optional string detail = 13;     // ルート情報など
  string hash = 14;                // 必須 - 重複チェック用ハッシュ
}
```

**重要な仕様:**
- `ic_fr` (入口IC) は **optional** - 実データの22.3%が空
- `date_to` は RFC3339形式 (例: `2025-10-18T15:30:00Z` または `2025-10-18 15:30:00`)
- `date_to_date` は YYYY-MM-DD形式 (例: `2025-10-18`)
- `hash` は自動生成されるため、送信時は空文字列でOK

**etc_data_processor側 (現在):**
```go
// processRecords内で作成されるデータ
dataToSave := map[string]interface{}{
    "account_id":   accountID,
    "date":        simpleRecord.Date,
    "entry_ic":    simpleRecord.EntryIC,
    "exit_ic":     simpleRecord.ExitIC,
    "route":       simpleRecord.Route,
    "vehicle_type": simpleRecord.VehicleType,
    "amount":      simpleRecord.Amount,
    "card_number": simpleRecord.CardNumber,
}
```

→ **フィールド名が完全に異なる**

## 修正案

### 方法1: DBClientをproto定義ベースにする（推奨）

#### 1.1 DBClientインターフェースを修正
```go
// src/pkg/handler/service.go

import (
    pb "github.com/yhonda-ohishi/db_service/src/proto"
)

type DBClient interface {
    // db_serviceのproto定義に合わせる
    CreateETCMeisai(ctx context.Context, req *pb.CreateETCMeisaiRequest) (*pb.ETCMeisaiResponse, error)
}
```

#### 1.2 processRecords関数を修正
```go
// src/pkg/handler/service.go の processRecords 関数

func (s *DataProcessorService) processRecords(ctx context.Context, records []parser.ActualETCRecord, accountID string, skipDuplicates bool) (*pb.ProcessingStats, []string) {
    // ... 既存のコード ...

    for i, record := range records {
        // ... バリデーション等 ...

        // Convert to simple format
        simpleRecord, err := s.parser.ConvertToSimpleRecord(record)
        if err != nil {
            errors = append(errors, fmt.Sprintf("Record %d: conversion failed: %v", i+1, err))
            stats.ErrorRecords++
            continue
        }

        // Parse dates
        exitDate, err := parseDate(simpleRecord.Date)
        if err != nil {
            errors = append(errors, fmt.Sprintf("Record %d: invalid date: %v", i+1, err))
            stats.ErrorRecords++
            continue
        }

        // Create request for db_service
        req := &pb.CreateETCMeisaiRequest{
            EtcMeisai: &pb.ETCMeisai{
                DateTo:     fmt.Sprintf("%s %s", exitDate.Format("2006-01-02"), simpleRecord.Time),
                DateToDate: exitDate.Format("2006-01-02"),
                IcFr:       simpleRecord.EntryIC,
                IcTo:       simpleRecord.ExitIC,
                Price:      int32(simpleRecord.Amount),
                Shashu:     int32(simpleRecord.VehicleType), // または適切な変換
                EtcNum:     simpleRecord.CardNumber,
                Detail:     &simpleRecord.Route,
            },
        }

        // Save to database via db_service
        if s.dbClient != nil {
            if _, err := s.dbClient.CreateETCMeisai(ctx, req); err != nil {
                errors = append(errors, fmt.Sprintf("Record %d: save failed: %v", i+1, err))
                stats.ErrorRecords++
                continue
            }
        }

        stats.SavedRecords++
    }

    return stats, errors
}
```

### 方法2: アダプターパターン（現状維持）

もし既存のインターフェースを維持したい場合:

#### 2.1 ETCMeisaiに変換する構造体を定義
```go
// src/pkg/handler/types.go (新規作成)

type ETCRecordForDB struct {
    EntryDate    string
    EntryTime    string
    ExitDate     string
    ExitTime     string
    EntryIC      string
    ExitIC       string
    Route        string
    VehicleClass int
    Amount       int
    CardNumber   string
}
```

#### 2.2 SaveETCDataの引数を明確化
```go
type DBClient interface {
    SaveETCData(record *ETCRecordForDB) error
}
```

## 推奨される修正

**方法1（DBClientをproto定義ベースにする）を推奨**

理由:
- 型安全性が高い
- db_serviceと直接連携できる
- マッピングエラーが起きにくい
- gRPCの利点を最大限活用できる

## 必要な変更ファイル

1. `src/pkg/handler/service.go`
   - `DBClient`インターフェースの修正
   - `processRecords`関数の修正

2. `src/go.mod`
   - db_serviceの依存関係追加
   ```
   require github.com/yhonda-ohishi/db_service v0.0.0-latest
   ```

3. `src/cmd/server/main.go`
   - DBClient初期化部分の修正例追加

## テスト方法

修正後、以下のコマンドでテスト:

```bash
# desktop-server側で
go run ./cmd/test-csv-import/main.go <csv_file_path> <account_id>
```

期待される結果:
- CSVファイルが正常にパース・DB登録される
- エラーが発生しない

## 参考情報

### db_service proto定義の場所

**ローカル:**
- `C:\go\db_service\src\proto\ryohi.proto`

**GitHub:**
- リポジトリ: `https://github.com/yhonda-ohishi/db_service`
- Proto定義: `https://github.com/yhonda-ohishi/db_service/src/proto/ryohi.proto`
- gRPCサービス定義: `ETCMeisaiService` (Create, Get, Update, Delete, List)

**Go package:**
```go
import pb "github.com/yhonda-ohishi/db_service/src/proto"
```

### 現在のetc_data_processor
- リポジトリ: `https://github.com/yhonda-ohishi/etc_data_processor`
- パッケージ: `github.com/yhonda-ohishi/etc_data_processor/src/pkg/handler`

### desktop-server統合コード
- リポジトリ: `https://github.com/yhonda-ohishi-pub-dev/desktop-server`
- 統合コード: `internal/etcdb/client.go`

## 質問・確認事項

1. SimpleETCRecordにExitDateとExitTimeが分かれて存在しますか？
2. VehicleTypeは文字列ですか、それとも数値ですか？
3. db_serviceのバージョンはいくつを想定していますか？

---

作成日: 2025-10-18
作成者: desktop-server integration team
