# 追加仕様 (2026-04-24) 設計ドキュメント

本文書は `temporary/additional_requirement042401.md` の全要件を整理し、DB・Backend・Frontend の変更内容を定義する。

---

## 要件と対応方針の一覧

| # | 要件概要 | 対応方針 | 影響レイヤ |
|---|---------|---------|-----------|
| R1 | Subscope (scope の下位区分) | 既存の `parent_scope_id` で対応済み。UI のツリー表示を強化 | Frontend |
| R2 | 各製品の増減 flow 確認 | 新規 API: Item Flow (inventory_events ベースの時系列表示) | Backend + Frontend |
| R3 | Snapshot: 日付指定による在庫予測 | 既存 snapshot API を拡張。target_date パラメータ追加 | Backend + Frontend |
| R4 | 各種一覧のフィルタリング強化 | item_number, manufacturer, category 等の検索。予測入力対応 | Backend + Frontend |
| R5 | Scope 概要一覧画面 | 新規画面: scope ツリー + requirements/reservations/shortage サマリカウント | Backend + Frontend |
| R6 | Shortage の時系列分解 (scope 開始前/後) | shortage API を拡張。開始日前後の入荷予定を分離表示 | Backend + Frontend |
| R7 | Item description 編集 | 既存対応済み (admin master-data items) | — |
| R8 | 未紐づけ Item の削除 | 既存対応済み (DELETE /api/v1/admin/master-data/items/{id}) | — |
| R9 | Reservation/Requirements/Shortage の CSV 出力 | shortage export は既存。reservation/requirements export を新規追加 | Backend + Frontend |
| R10 | Requirements の CSV 入力 | 新規 API: requirements CSV import (preview + apply) | Backend + Frontend |
| R11 | Requirements から直接 reservation (将来入荷も考慮) | 新規 API: bulk reservation preview + confirm | DB + Backend + Frontend |
| R12 | Reservation と Order の紐づけ | reservation_allocations に purchase_order_line_id, source_type 追加 | DB + Backend |
| R13 | Scope 充当部品の Order 情報確認 | shortage/scope 詳細画面で PO line 情報を表示 | Backend + Frontend |
| R14 | Requirements 登録時の未登録 item inline 登録 | requirements CSV import 中に item 登録導線を提供 | Frontend |
| R15 | Item number 表示時の manufacturer/description ポップオーバー | 新規共通コンポーネント: ItemInfoPopover | Frontend |
| R16 | Arrival カレンダー | 新規画面: PO line の expected_arrival_date をカレンダー表示 | Backend + Frontend |
| R17 | 一覧の collapse/filtering 機能 | 全リスト画面にテキスト検索・折りたたみを追加 | Frontend |

---

## 1. DB Schema 変更 (Migration 000011)

### 1.1 reservation_allocations 拡張

```sql
ALTER TABLE reservation_allocations
    ADD COLUMN IF NOT EXISTS purchase_order_line_id TEXT
        REFERENCES purchase_order_lines(id) ON DELETE SET NULL,
    ADD COLUMN IF NOT EXISTS source_type TEXT NOT NULL DEFAULT 'stock';
-- source_type: 'stock' | 'incoming_order'
```

### 1.2 フィルタリング用インデックス

```sql
-- Item 検索の高速化
CREATE INDEX IF NOT EXISTS idx_items_canonical_item_number_trgm
    ON items USING gin (canonical_item_number gin_trgm_ops);
CREATE INDEX IF NOT EXISTS idx_items_description_trgm
    ON items USING gin (description gin_trgm_ops);

-- Inventory events の item/日付検索
CREATE INDEX IF NOT EXISTS idx_inventory_events_item_occurred
    ON inventory_events (item_id, occurred_at DESC);

-- PO lines の arrival date 検索
CREATE INDEX IF NOT EXISTS idx_po_lines_expected_arrival
    ON purchase_order_lines (expected_arrival_date)
    WHERE expected_arrival_date IS NOT NULL AND status NOT IN ('cancelled', 'received');

-- Scope item requirements の scope 検索
CREATE INDEX IF NOT EXISTS idx_scope_item_requirements_scope
    ON scope_item_requirements (scope_id);
```

**注:** `pg_trgm` 拡張が必要。Migration 冒頭で `CREATE EXTENSION IF NOT EXISTS pg_trgm;` を実行する。

---

## 2. Backend API 変更

### 2.1 新規エンドポイント

| Method | Path | 説明 |
|--------|------|------|
| GET | `/api/v1/inventory/items/{id}/flow` | Item の増減フロー (時系列イベント一覧) |
| GET | `/api/v1/inventory/snapshot` | 既存拡張: `target_date` パラメータ追加 |
| GET | `/api/v1/operator/scope-overview` | Scope 概要一覧 (サマリカウント付き) |
| GET | `/api/v1/operator/shortages/timeline` | Shortage の時系列分解 |
| GET | `/api/v1/operator/reservations/export` | Reservation CSV 出力 |
| GET | `/api/v1/operator/requirements/export` | Requirements CSV 出力 |
| POST | `/api/v1/operator/requirements/import/preview` | Requirements CSV import preview |
| POST | `/api/v1/operator/requirements/import` | Requirements CSV import apply |
| POST | `/api/v1/operator/reservations/bulk-preview` | Requirements からの一括 reservation preview |
| POST | `/api/v1/operator/reservations/bulk-confirm` | 一括 reservation 確定 |
| GET | `/api/v1/inventory/arrivals/calendar` | 到着カレンダーデータ |
| GET | `/api/v1/admin/master-data/items/suggest` | Item 予測入力 (typeahead) |
| GET | `/api/v1/admin/master-data/categories/suggest` | Category 予測入力 |

### 2.2 既存エンドポイント拡張

#### GET `/api/v1/inventory/snapshot`
- 新規パラメータ: `target_date` (YYYY-MM-DD)
- target_date 指定時の計算:
  1. 現在の inventory_balances (on_hand)
  2. \+ expected_arrival_date <= target_date の PO lines の未入荷分
  3. \- active reservations の確保分
  4. location 別に集計

#### GET `/api/v1/operator/shortages`
- 新規パラメータ: `coverage_rule` (none | submitted | approved | ordered | received)
- 新規レスポンスフィールド:
  - `in_request_flow` (procurement status projection から)
  - `ordered_quantity`
  - `received_quantity`
  - `actionable_shortage`
  - `related_procurement_request_ids`

#### 全一覧系 API
- 共通パラメータ追加: `q` (テキスト検索), `manufacturer`, `category`, `item_number`
- 既存の device/scope フィルタと併用可能

### 2.3 レスポンスモデル

#### ItemFlow

```go
type ItemFlowEntry struct {
    Date          string // YYYY-MM-DD
    EventType     string // receive, consume, reserve_allocate, etc.
    QuantityDelta int
    Balance       int    // running balance after this event
    SourceType    string // purchase_order, reservation, manual, etc.
    SourceRef     string // PO number, reservation ID, etc.
    Note          string
}
```

#### ScopeOverviewRow

```go
type ScopeOverviewRow struct {
    DeviceKey            string
    DeviceName           string
    ScopeID              string
    ScopeKey             string
    ScopeName            string
    ScopeType            string
    ParentScopeID        string
    Status               string
    PlannedStartAt       string
    RequirementsCount    int
    ReservationsCount    int
    ShortageItemCount    int // actionable shortage > 0 の品目数
    OwnerDepartment      string
}
```

#### ShortageTimelineEntry

```go
type ShortageTimelineEntry struct {
    ItemID              string
    ItemNumber          string
    Manufacturer        string
    Description         string
    RequiredQuantity    int
    AvailableByStart    int    // scope 開始日までに使用可能な数
    DelayedArrivals     []DelayedArrival
    ShortageAtStart     int    // required - availableByStart
}

type DelayedArrival struct {
    ExpectedDate        string
    Quantity            int
    PurchaseOrderNumber string
    PurchaseOrderLineID string
}
```

#### BulkReservationPreview

```go
type BulkReservationPreviewRow struct {
    ItemID              string
    ItemNumber          string
    Manufacturer        string
    Description         string
    RequiredQuantity    int
    AllocFromStock      int
    AllocFromStockLocs  []StockAllocation
    AllocFromOrders     int
    AllocFromOrderLines []OrderAllocation
    Unallocated         int
}

type StockAllocation struct {
    LocationCode string
    Quantity     int
}

type OrderAllocation struct {
    PurchaseOrderLineID string
    PurchaseOrderNumber string
    ExpectedArrival     string
    Quantity            int
}
```

#### ArrivalCalendarEntry

```go
type ArrivalCalendarEntry struct {
    Date                 string
    Items                []ArrivalItem
}

type ArrivalItem struct {
    ItemID               string
    ItemNumber           string
    Manufacturer         string
    Description          string
    Quantity             int
    PurchaseOrderNumber  string
    PurchaseOrderLineID  string
    QuotationNumber      string
    SupplierName         string
}
```

---

## 3. Frontend 変更

### 3.1 新規画面

| 画面 | ルート | 説明 |
|------|--------|------|
| Scope 概要 | `/app/operator/scopes` | Scope ツリー + サマリカウント表示。クリックで詳細遷移 |
| Item Flow | `/app/inventory/items/{id}/flow` | 品目別の増減タイムライン |
| Arrival Calendar | `/app/inventory/arrivals/calendar` | 月表示カレンダー。日付クリックで詳細 |

### 3.2 新規共通コンポーネント

#### ItemInfoPopover
- item_number テキスト上にホバーまたはクリックで表示
- 内容: manufacturer, description, category, lifecycle_status
- Popover コンポーネントベース

#### CollapsibleFilterBar
- テキスト検索フィールド
- manufacturer / category / status のドロップダウンフィルタ
- 折りたたみ可能なフィルタパネル
- 全一覧画面で共用

#### ScopeTreeView
- device_scopes の parent_scope_id に基づくツリー表示
- 各ノードにサマリカウントバッジ
- クリックで scope 選択 → context bar 更新

### 3.3 既存画面の拡張

#### ShortagesPage
- フィルタ追加: coverage_rule セレクタ
- カラム追加: In Request Flow, Ordered, Received, Actionable Shortage
- 行クリックで関連 procurement request へのリンク
- Shortage Timeline 表示 (scope 開始日前後の分解)

#### ReservationsPage
- CSV Export ボタン追加
- source_type 列追加 (stock / incoming_order)
- 紐づけ PO 番号の表示 (ItemInfoPopover 類似のポップオーバー)

#### OperatorDashboardPage (Requirements)
- CSV Export / Import ボタン追加
- "一括 Reservation" ボタン追加
- Import 時の未登録 item 検知 → inline 登録ダイアログ

#### InventoryPage
- Snapshot に日付ピッカー追加
- target_date 指定時は予測値として表示 (視覚的に区別)

#### 全一覧画面
- CollapsibleFilterBar 統合
- テキスト検索対応

### 3.4 新規 Hooks

```typescript
useItemFlow(itemId: string)
useScopeOverview(device?: string)
useShortageTimeline(device: string, scope: string)
useArrivalCalendar(yearMonth: string)
useItemSuggest(query: string)
useCategorySuggest(query: string)
```

---

## 4. 実装順序

1. **DB Migration** (000011): reservation_allocations 拡張 + インデックス
2. **Backend Models**: 新規レスポンス型の定義
3. **Backend Repository**: 新規クエリ実装
4. **Backend Service**: 新規サービスメソッド
5. **Backend Handlers + Router**: 新規エンドポイント登録
6. **Frontend Hooks**: 新規データ取得フック
7. **Frontend Components**: ItemInfoPopover, CollapsibleFilterBar, ScopeTreeView
8. **Frontend Pages**: Scope 概要, Item Flow, Arrival Calendar
9. **Frontend 既存画面拡張**: Shortage, Reservations, Requirements, Inventory

---

## 5. 設計判断の根拠

### Reservation-Order 紐づけ: Allocation 単位

reservation_allocations テーブルに `purchase_order_line_id` と `source_type` を追加する方式を採用。理由:
- 1 つの reservation が在庫と入荷予定の両方から割り当てられるケースに自然に対応
- 部分入荷時に allocation 行を分割することで正確な追跡が可能
- 既存の allocation/release フローとの整合性が高い

### Snapshot: 在庫 + 発注入荷予定 - 予約確保

requirements を含めない理由:
- requirements は計画段階の数値であり、変動が大きい
- shortage 画面で requirements との差分は別途確認できる
- snapshot は「物理的に使える見込み数」に集中させる

### Scope 概要: サマリカウント

詳細プログレスではなくカウント表示とする理由:
- 一覧の API 負荷を抑える
- scope 数が増えても一覧性を維持
- 詳細はクリック遷移で確認するフロー
