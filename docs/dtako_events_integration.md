# dtako_events統合ドキュメント

## 概要

desktop-serverに`dtako_events`サービスを統合し、フロントエンドからgRPC-Web経由でアクセス可能にしました。

## 統合日時

2025-10-20

## アーキテクチャ

```
Frontend → desktop-server (gRPC-Web: http://localhost:8080/api/) → dtako_events services
```

desktop-serverと同一プロセスで動作します（db_serviceやetc_meisai_scraperと同様）。

## 実装ファイル

### 追加ファイル
- `server/dtako_service.go` - dtako_eventsサービスの実装ファイル

### 変更ファイル
- `server/grpc.go` (L38-L43) - サービス登録
- `server/db.go` (L101-L103) - QueryRowメソッド追加
- `go.mod` - `github.com/yhonda-ohishi/dtako_events v0.2.1` 依存関係

## 登録されたサービス

### 1. dtako.DtakoRowService (運行データ管理)

**パッケージ**: `github.com/yhonda-ohishi/dtako_events/proto`
**サービス名**: `dtako.DtakoRowService`

#### メソッド一覧

| メソッド | リクエスト | レスポンス | 説明 | 実装状態 |
|---------|-----------|-----------|------|---------|
| GetRowDetail | GetRowDetailRequest | GetRowDetailResponse | 運行データの詳細取得（view機能） | ❌ Unimplemented |
| CreateRow | CreateRowRequest | Row | 運行データ作成 | ❌ Unimplemented |
| GetRow | GetRowRequest | Row | 運行データ取得 | ❌ Unimplemented |
| UpdateRow | UpdateRowRequest | Row | 運行データ更新 | ❌ Unimplemented |
| DeleteRow | DeleteRowRequest | DeleteRowResponse | 運行データ削除 | ❌ Unimplemented |
| ListRows | ListRowsRequest | ListRowsResponse | 運行データ一覧取得 | ❌ Unimplemented |
| SearchRows | SearchRowsRequest | ListRowsResponse | 運行データ検索 | ❌ Unimplemented |
| SearchByShaban | ShabanSearchRequest | ListRowsResponse | 車番で検索 | ❌ Unimplemented |

### 2. dtako.DtakoEventService (イベントデータ管理)

**パッケージ**: `github.com/yhonda-ohishi/dtako_events/proto`
**サービス名**: `dtako.DtakoEventService`

#### メソッド一覧

| メソッド | リクエスト | レスポンス | 説明 | 実装状態 |
|---------|-----------|-----------|------|---------|
| CreateEvent | CreateEventRequest | Event | イベント作成 | ❌ Unimplemented |
| GetEvent | GetEventRequest | Event | イベント取得 | ❌ Unimplemented |
| UpdateEvent | UpdateEventRequest | Event | イベント更新 | ❌ Unimplemented |
| DeleteEvent | DeleteEventRequest | DeleteEventResponse | イベント削除 | ❌ Unimplemented |
| ListEvents | ListEventsRequest | ListEventsResponse | イベント一覧取得 | ❌ Unimplemented |
| FindEmptyLocation | FindEmptyLocationRequest | ListEventsResponse | 空の位置情報を検索 | ❌ Unimplemented |
| SearchByDateRange | DateRangeRequest | ListEventsResponse | 日付範囲で検索 | ❌ Unimplemented |
| SearchByDriver | DriverSearchRequest | ListEventsResponse | 運転手で検索 | ❌ Unimplemented |
| SetLocationByGeo | SetLocationRequest | SetLocationResponse | 位置情報を地理座標で設定 | ❌ Unimplemented |
| SetGeoCode | SetGeoCodeRequest | SetGeoCodeResponse | ジオコード設定 | ❌ Unimplemented |

## フロントエンドでの使用方法

### エンドポイント
- **URL**: `http://localhost:8080/api/`
- **プロトコル**: gRPC-Web

### 使用例

```javascript
// DtakoRowServiceクライアント
import { DtakoRowServiceClient } from './generated/dtako_rows_grpc_web_pb';
import { GetRowDetailRequest } from './generated/dtako_rows_pb';

const client = new DtakoRowServiceClient('http://localhost:8080/api');

const request = new GetRowDetailRequest();
request.setId("row-123");

client.getRowDetail(request, {}, (err, response) => {
  if (err) {
    console.error('Error:', err);
    // 現在は "Unimplemented" エラーが返る
    return;
  }

  const row = response.getDtakoRow();
  const events = response.getEventsList();
  const pairs = response.getTsumiOroshiPairsList();

  console.log('Row:', row);
});
```

```javascript
// DtakoEventServiceクライアント
import { DtakoEventServiceClient } from './generated/dtako_events_grpc_web_pb';
import { ListEventsRequest } from './generated/dtako_events_pb';

const eventClient = new DtakoEventServiceClient('http://localhost:8080/api');

const request = new ListEventsRequest();
request.setPage(0);
request.setPageSize(10);

eventClient.listEvents(request, {}, (err, response) => {
  if (err) {
    console.error('Error:', err);
    // 現在は "Unimplemented" エラーが返る
    return;
  }

  const events = response.getEventsList();
  console.log('Events:', events);
});
```

## 現在の実装状態

### ✅ 完了
- dtako_events v0.2.1の依存関係追加
- DtakoRowServiceの登録
- DtakoEventServiceの登録
- gRPC-Web経由でのアクセス可能
- QueryRowメソッドの追加

### ❌ 未実装（Unimplemented）
- 全16メソッドの実装（上記表参照）
- データベースアクセスロジック
- ビジネスロジック

### 実装方法

実装は`server/dtako_service.go`に追加します：

```go
package server

import (
	"context"
	dtakopb "github.com/yhonda-ohishi/dtako_events/proto"
)

// 現在の実装
type DtakoRowServiceImpl struct {
	dtakopb.UnimplementedDtakoRowServiceServer
	db *DatabaseConnection
}

// 実装例
func (s *DtakoRowServiceImpl) GetRowDetail(ctx context.Context, req *dtakopb.GetRowDetailRequest) (*dtakopb.GetRowDetailResponse, error) {
	// TODO: データベースから運行データを取得
	// TODO: イベントデータを取得
	// TODO: 積み降しペアを取得
	return &dtakopb.GetRowDetailResponse{
		DtakoRow: &dtakopb.Row{...},
		Events: []*dtakopb.Event{...},
		TsumiOroshiPairs: []*dtakopb.TsumiOroshiPair{...},
	}, nil
}
```

## エラーハンドリング

現在、未実装のメソッドを呼び出すと以下のエラーが返ります：

```
code = Unimplemented
desc = method GetRowDetail not implemented
```

これはgRPCの標準的な動作で、`UnimplementedDtakoRowServiceServer`と`UnimplementedDtakoEventServiceServer`を埋め込んでいるためです。

## データベーステーブル構造（想定）

### dtako_rows テーブル
- `id` - 主キー
- `unko_no` - 運行NO
- `shaban` - 車番
- `driver_id` - 運転手ID
- `start_datetime` - 出庫日時
- `end_datetime` - 帰庫日時
- `distance` - 走行距離
- `fuel_used` - 燃料使用量
- `created_at` - 作成日時
- `updated_at` - 更新日時

### dtako_events テーブル
- `srch_id` - 主キー
- `event_type` - イベントタイプ（TSUMI/OROSHI等）
- `unko_no` - 運行NO
- `driver_id` - 運転手ID
- `start_datetime` - 開始日時
- `end_datetime` - 終了日時
- `start_latitude` - 開始緯度
- `start_longitude` - 開始経度
- `start_city_name` - 開始地点都市名
- `end_latitude` - 終了緯度
- `end_longitude` - 終了経度
- `end_city_name` - 終了地点都市名
- `tokuisaki` - 得意先
- `biko` - 備考
- `created_at` - 作成日時
- `updated_at` - 更新日時

## 関連リンク

- dtako_eventsリポジトリ: https://github.com/yhonda-ohishi/dtako_events
- Protocol Buffers定義: `github.com/yhonda-ohishi/dtako_events/proto`
- 実装ファイル: [server/dtako_service.go](../server/dtako_service.go)
- gRPC登録: [server/grpc.go](../server/grpc.go#L38-L43)

## 次のステップ

1. データベーステーブルの作成（dtako_rows, dtako_events）
2. 各メソッドの実装
3. ユニットテストの追加
4. フロントエンドからの動作確認

## 注意事項

- protoファイルの変更があった場合は、dtako_eventsリポジトリで`protoc`を実行してコードを再生成し、新しいバージョンをリリースする必要があります
- desktop-serverは生成されたprotoファイルを使用するだけで、protoファイル自体は保持しません
- 実装は別途追加する予定（現在はUnimplemented）
