# 事前実装計画

## 目的

外部連携 API（integration web server / 楽楽販売 / バクラク）の最終契約が未確定な状態でも、
このアプリケーション単体で先行実装できる範囲を明確にし、手戻りを抑えながら主要業務フローを早期に成立させる。

本計画では、**Local mode で主要フローを通すことを優先**しつつ、**Cloud mode / Auth / CI/CD を後から無理なく載せられる骨組み**を最初から入れる方針を採る。

到達目標は、単にローカルで業務フローが動くことではなく、**remote repository (`https://github.com/Kenshin-Isogai/inventory_manager.git`) への push を起点に GitHub Actions で Docker image を build し、Cloud Run へ継続的に deploy できる状態**まで含める。

---

## 現時点で確定している前提

1. `GUI` は実質的に this application 全体（frontend + backend）を指す。
2. 外部システムとの通信は基本的に backend 経由とする。
3. webhook の受け口は this application の backend とする。
4. webhook 受信後、backend が必要に応じて外部へ再取得を行い、ローカル projection を更新する。
5. 見積書 OCR は **Gemini on Vertex AI** で PDF / 画像を解析する前提で進める。
6. 外部 API 契約が未確定の箇所は adapter / port に隔離し、モック実装で先行する。
7. `Operator / Inventory / Procurement / Admin` は主に **アプリ区分 / 画面区分** であり、権限制御そのものとは分けて扱う。
8. `Acceptance Inspector` は独立ロールとして扱い、コード上や DB 上の role key は `receiving_inspector` に統一する。
9. `Device / Scope` 文脈は全画面必須ではなく、部品管理・部品割当・申請区分など対象業務でのみ必須または強く推奨とする。
10. shortage 画面は procurement request を直接作成する主導線ではなく、**不足部品リストの確認・CSV 出力・既存 request 確認** を中心とする。
11. 楽楽販売を掌理する管理部は、このアプリケーションの `admin` role とは別概念として扱う。
12. 見積書 submit payload には、少なくとも **quotation PDF** と **構造化データ** が含まれる。
13. 構造化データの最終 format（JSON / CSV など）は未確定だが、既知フィールドは先に内部モデルへ反映してよい。
14. 楽楽販売 API は基本的に **HTTPS POST + UTF-8** で通信し、URL 形式は `https://{domain}/{account}/api/{api_name}/version/{api_version}` とする。
15. 楽楽販売 API の認証は `X-HD-apitoken` ヘッダで行う。
16. 楽楽販売 API は基本的に JSON レスポンスを返し、`status / code / items / errors / accessTime` を共通的に扱う。
17. 楽楽販売 API には通常 1 分あたり 20 回のリクエスト制限があり、超過時は `429` を返す前提で設計する。
18. 証憑ファイル ID の採番責務は楽楽販売側が持ち、このアプリケーションは同期済み ID を参照する。
19. project / budget category マスタは、更新時 webhook を起点に this application 側 cache を更新する前提で設計する。
20. ただし webhook の取りこぼしに備え、手動再同期または定期 reconcile を fallback として残す。

---

## 開発方針

## 1. 先に固めるもの

* ドメインモデル
* DB スキーマと migration
* backend のレイヤ構造
* frontend のルーティングと画面骨格
* frontend の画面区分とロール境界
* frontend / backend 間のローカル API 契約
* 認証・認可の差し込み位置
* Device / Scope 文脈の適用ルール
* OCR ジョブと購買申請フローの内部状態管理
* 見積書 submit payload の内部正規化モデル
* webhook 受信と projection 更新の内部フロー
* Docker build / runtime contract
* GitHub Actions から Cloud Run へ deploy する CI/CD 導線
* Cloud Run 運用に必要な設定値と secret 注入方式

## 2. 後から差し替えるもの

* integration web server 向け送信 payload
* 楽楽販売 / バクラクの API 詳細
* 外部 status の raw vocabulary
* 外部 master 同期の細かな取得契約
* 証憑ファイルの最終的な外部 ID / 保存責務詳細
* 見積書 structured payload の最終 serialization format
* 楽楽販売 API 名ごとの具体 request / response schema
* Cloud Run 上でのみ確定するスケール値や運用閾値

## 3. 設計原則

* 外部連携なしでもアプリ内部のユースケースが完結するようにする
* 画面は外部 API ではなく backend のローカル API だけを見る
* アプリ区分（Operator / Inventory / Procurement / Admin）と権限ロール（admin / operator / auditor / receiving_inspector など）を混同しない
* 画面遷移や表示制御は、アプリ区分・ロール・現在の Device / Scope 文脈の 3 軸で決める
* shortage / procurement status は projection を参照し、画面表示時に外部へ直接問い合わせない
* OCR 結果は常に draft 扱いとし、ユーザー確定値を正とする
* submit payload は PDF とユーザー確定済み structured data の組で扱う
* 外部同期まわりは append-only history と current projection を分ける
* 楽楽販売 adapter は HTTP 通信詳細を閉じ込め、usecase 層には正規化済み結果だけを返す
* deploy 可能性を後付けにしないため、backend / frontend ともにコンテナ実行前提の entrypoint・設定注入方式を早期固定する
* CI/CD は branch push で再現可能であることを重視し、手元専用の起動手順に依存しない

---

## 実装フェーズ

## Phase 0: プロジェクト土台の整備

### 目的

今後の実装を載せるための共通骨格を作る。

### 範囲

* backend の標準構成整理
  * HTTP handler
  * middleware
  * usecase
  * repository
  * external adapters
* 設定管理
  * local / cloud 切り替え
  * auth mode
  * storage mode
* migration 基盤
* PostgreSQL 接続基盤
* 監査ログ / 構造化ログの土台
* health / ready / api health
* storage interface
  * local filesystem
  * cloud storage 用の拡張ポイント
* Dockerfile / .dockerignore / コンテナ起動 contract
* Cloud Run を意識した port / health check / stateless 実行前提
* frontend 初期構成
  * Vite + React + TypeScript
  * ルーティング
  * SWR
  * レイアウト / 共通ナビゲーション
  * app shell / auth shell
  * route guard
  * global context bar と URL / state 設計
* frontend / backend 契約の先行定義
  * 画面別 read model API
  * command API
  * fixture / mock data
* テスト土台
  * router / auth / role visibility の最小テスト
  * モック API での画面疎通確認
* CI 土台
  * lint / test / build の GitHub Actions job
  * Docker build 検証 job
  * main branch からの deploy workflow 雛形

### この段階での完成条件

* backend / frontend がそれぞれ起動できる
* DB migration を独立実行できる
* Local mode の基本起動経路が成立する
* Docker build がローカルまたは CI 上で再現可能
* Cloud Run に必要な runtime env を設定経由で差し込める構造になっている
* auth / OCR / 外部連携を差し込む位置がコード上で明確になっている
* アプリ区分、ロール、Device / Scope 文脈の責務分離がコード上で明確になっている
* 主要画面のモック API 契約が固定されている

---

## Phase 1: 基幹ドメインと DB 実装

### 目的

外部連携に依存しない主要業務ロジックを先に成立させる。

### 優先対象

* master / catalog
  * items
  * suppliers
  * manufacturers
  * departments
  * categories
* device / scope
* inventory
  * balances
  * events
  * undo
* reservations
* shortage 計算
* users / roles / user_roles
* audit_events
* imports の内部モデル
* item master / alias 登録フロー

### 進め方

* まず migration と repository を作る
* 次に usecase を作る
* 最後に画面用 API を生やす
* 同時に shortage / reservation / inventory / import の read model を固定する

### この段階での完成条件

* 在庫操作、予約、不足算出、import 履歴確認がローカル DB だけで成立する
* Operator / Inventory / Admin の基礎画面が backend API で表示できる

---

## Phase 2: Procurement の内部モデル先行実装

### 目的

外部実行主体は別でも、this application 側の申請・追跡責務を完成させる。

### 範囲

* procurement_batches
* procurement_lines
* purchase_orders / purchase_order_lines
* procurement_status_projections
* procurement_status_history
* external_projects
* external_project_budget_categories
* supplier_quotations / quotation_lines

### 実装内容

* 購買要求の内部作成 API
* project / budget category のローカル参照 API
* 申請一覧 / 詳細 API
* normalized status の内部 vocabulary 定義
* OCR で取り込んだ見積書から procurement request を作成する流れ
* shortage 画面から関連 procurement request を参照する流れ
* quotation draft / submission payload の正規化
* 外部未連携時でも追跡可能なローカル状態遷移

### この段階での完成条件

* 外部送信なしでも procurement request の作成・一覧・詳細・内部追跡ができる
* shortage 画面から不足 CSV 出力と関連 request 参照ができる

---

## Phase 3: OCR 連携

### 目的

見積書アップロードから OCR preview/edit までを先行実装する。

### 範囲

* ファイルアップロード
* artifact 保存
* OCR ジョブ作成
* Gemini on Vertex AI 呼び出し
* OCR 結果保存
* OCR preview/edit API
* 再試行
* 失敗時の状態管理
* item 未登録時の登録導線
* item / alias CSV import

### 実装方針

* frontend から PDF / 画像を backend へ送信
* backend が artifact を保存
* backend が Gemini on Vertex AI を呼び出す
* OCR 結果は draft として保持
* ユーザー編集後の確定値のみ procurement request 作成に使う
* item が master 未登録の行は、submission 前に必ず item 登録を完了させる
* item 登録は権限承認なしで実行可能とし、申請画面から遷移または inline で処理できるようにする
* item 登録では少なくとも以下を入力対象とする
  * 必須
    * manufacturer
    * canonical item number
    * description
    * item category
  * 推奨
    * default supplier
    * note
* alias は item 本体登録と分けて扱ってよいが、quotation review 中に必要なら同一フロー内で追加できるようにする
* pack 品番など canonical item と異なる見積記載品番は `supplier_item_aliases` で表現する
  * 例: canonical item `ER2`
  * supplier-specific ordered item number `ER2-P4`
  * `units_per_order = 4`
* submit payload は以下の既知項目を保持する
  * quotation 単位
    * supplier
    * quotation number
    * issue date
  * row 単位の OCR 抽出値
    * manufacturer
    * item number
    * item description
    * quantity
    * leadtime
  * row 単位のユーザー入力値
    * delivery location
    * budget category
    * accounting category
    * supplier contact（supplier 未登録時のみ）
* leadtime は expected arrival date ではなく、**order date から到着予定日までの duration** として扱う
* shortage CSV の既知列は少なくとも以下とする
  * device
  * scope
  * manufacturer
  * item number
  * description
  * quantity

### この段階での完成条件

* 見積書アップロード
* OCR 実行
* 抽出結果プレビュー
* ユーザー編集
* project / budget category 選択
* delivery location / accounting category / supplier contact 入力
* item / alias の登録または CSV 事前投入
* request 作成直前までの UX

---

## Phase 4: 外部連携境界の固定

### 目的

未確定 API の影響範囲を閉じ込めつつ、実装を前に進める。

### 範囲

* integration web server client interface
* 楽楽販売 reconciliation client interface
* master sync interface
* webhook handler
* projection updater
* mock adapters
* retry / backoff policy

### adapter で固定する契約

* procurement request submit
* reconciliation fetch
* project master fetch
* budget category fetch
* webhook authenticity verification
* quotation PDF + structured payload submit
* API error normalization
* rate-limit aware retry
* evidence file id reference sync
* webhook-driven master cache refresh

### integration adapter interface 草案

backend の usecase 層からは、少なくとも以下のような抽象契約で扱える状態を目標とする。

* `submit_procurement_request(input)`
  * 入力
    * quotation PDF artifact 参照
    * structured payload
    * idempotency key
  * 出力
    * external request reference
    * evidence file references
    * accepted_at
    * raw response
* `fetch_procurement_reconciliation(input)`
  * 入力
    * external request reference または同期対象キー
  * 出力
    * normalized procurement state
    * raw external statuses
    * quantity progression
    * observed_at
    * raw response
* `fetch_project_master()`
  * 出力
    * project rows
    * synced_at
    * raw response
* `fetch_budget_categories(project_key)`
  * 出力
    * budget category rows
    * synced_at
    * raw response
* `verify_webhook(input)`
  * 入力
    * headers
    * body
  * 出力
    * accepted / rejected
    * normalized event metadata
* `normalize_api_error(input)`
  * 出力
    * retryable
    * normalized code
    * user-facing correction hint
    * audit payload

### 楽楽販売 API の既知通信仕様

* 基本通信方式
  * HTTPS POST
  * UTF-8
  * URL 形式: `https://{domain}/{account}/api/{api_name}/version/{api_version}`
* 基本リクエストヘッダ
  * `X-HD-apitoken: {api_token}` を必須
  * 多くの API では `Content-Type: application/json; charset=utf-8`
  * file upload など一部 API では別 Content-Type を許容する前提で adapter を分岐可能にする
* 基本レスポンス
  * `status`
  * `code`
  * `items`
  * `errors`
  * `accessTime`
* 基本エラー処理
  * `200` 成功
  * `400` 入力エラー
  * `401` 認証エラー
  * `429` レート制限超過
  * 入力エラー時は `errors` 内のコード・メッセージ・項目詳細を保持して UI / 監査ログへ渡せる形に正規化する

### この段階での完成条件

* backend は mock adapter で end-to-end に近い動作ができる
* 将来の実 API 実装は adapter 差し替えで済む構造になる
* `429` や `401` を含む外部 API エラーを内部エラー型へ正規化できる
* 証憑ファイル ID を外部参照値として保持・再表示できる
* project / budget category master を webhook 起点で更新しつつ、fallback 再同期も行える
* shortage CSV / item CSV / alias CSV の運用がローカル検証できる

---

## Phase 5: 認証・認可の骨組み

### 目的

主要フロー優先を維持しつつ、Auth を後付けではなく最初から差し込める形にする。

### 範囲

* JWT middleware
* AUTH_MODE / RBAC_MODE
* local/dev 用 bearer token
* OIDC/JWKS 用の interface
* users との identity resolution
* self-registration / approval 用の土台 API
* frontend 側の auth state / route guard の土台
* role-permission matrix

### この段階での完成条件

* `none` / `dry_run` / `enforced` を切り替え可能
* ローカル優先で開発しながら、本番向け auth を差し込める
* Cloud Run 上で利用する identity / secret 注入方式に矛盾しない

---

## Phase 6: Frontend 業務画面の統合

### 目的

仕様書にある役割別アプリ構成を、実際の SPA としてつなぐ。

### 優先画面

1. Operator
   * Requirements
   * Reservations
   * Shortage
   * Imports Upload / History
2. Inventory
   * by item
   * by location
   * receive / move / adjust
3. Procurement
   * request list
   * request detail
   * request creation
   * OCR queue
4. Admin
   * users
   * roles
   * items / suppliers / aliases

### この段階での完成条件

* Local mode で主要フローが UI 上で通る
* shortage 確認 / CSV 出力、OCR 取込、request 作成、追跡まで一連の画面遷移ができる
* item / alias を事前または申請途中に登録できる

---

## Phase 7: 同期・通知・運用補強

### 目的

外部イベント反映と運用安定性を高める。

### 範囲

* webhook 受信
* reconciliation 実行
* procurement projection 更新
* status history 追記
* 通知トリガ
* 障害時の再実行導線
* 監査ログ拡充

### この段階での完成条件

* webhook を起点に projection が更新される
* 一覧 / 詳細が local projection ベースで更新される
* 再同期 / 再試行 / 最終同期時刻が UI 上で確認できる

---

## Phase 8: CI/CD・Cloud Run デプロイ整備

### 目的

remote repository を起点に、手作業なしで build / test / deploy を回せる状態を作る。

### 範囲

* backend / frontend の本番 Docker build 固定
* 必要なら frontend 配信方式の固定
  * backend から静的配信するか
  * frontend を別 service とするか
* Cloud Run deploy 用設定
  * service 名
  * region
  * image repository
  * 実行 service account
  * min/max instances
  * concurrency
  * ingress
* Artifact Registry への image push
* GitHub Actions workflow
  * pull request 用 CI
  * main branch merge/push 用 deploy
* migration 実行方式の固定
  * deploy 前 job で実行するか
  * 起動時に安全に実行するか
  * 手動運用に切り出すか
* Cloud Run / GCP 用 secret と env の棚卸し
* deploy rollback の最小運用手順

### この段階での完成条件

* GitHub Actions 上で test / build / docker build が通る
* `main` への push で Artifact Registry へ image が publish される
* 同 workflow から Cloud Run service が更新される
* deploy 成功後に health endpoint で疎通確認できる
* 必要な secret / env / service account 権限がドキュメント化されている
* 少なくとも staging 相当 1 環境に対して継続 deploy できる

### 実装メモ

* GitHub Actions から GCP 認証する方式は、長期鍵よりも Workload Identity Federation を優先する
* Cloud Run は stateless 前提とし、永続化は PostgreSQL / Cloud Storage / 外部サービスへ逃がす
* DB migration は deploy pipeline に組み込む場合でも idempotent で rollback 方針を持つ
* frontend を同一 service に載せる場合は SPA fallback と API routing を Docker image / app server 側で固定する
* frontend と backend を分離 deploy する場合は、CORS・認証 cookie / token・環境ごとの API base URL を明示管理する

---

## 先行実装で到達できる状態

外部 API の最終契約が未確定でも、以下までは十分に進められる。

* 在庫・予約・不足算出の完成
* 購買 request の内部モデル完成
* OCR を含む request 起票 UX の完成
* shortage CSV 出力と import / export の基本運用
* project / budget category のローカルキャッシュ参照
* webhook 受信から projection 更新までの内部フロー
* mock adapter を使った end-to-end 検証
* auth / cloud 対応の骨組み
* Docker 化された本番実行形の固定
* GitHub Actions から Cloud Run への継続 deploy の成立

言い換えると、**アプリ本体の大部分は先行実装可能**であり、後で差し替えるのは主に外部接続 adapter とそのマッピングである。加えて、**deploy 導線そのものも先行で固められる**ため、実 API 契約確定後の本番反映までのリードタイムを短くできる。

---

## 現時点の未確定事項

以下は早めに詰めたいが、未確定でも当面は adapter 境界に閉じ込められる。

1. integration web server への submit payload 詳細
2. webhook の認証方式と署名仕様
3. 楽楽販売からの reconciliation GET のレスポンス形式
4. external status -> normalized status の最終マッピング
5. バクラク由来情報のどこまでを this application が保持するか
6. structured payload の最終 serialization format（JSON / CSV / multipart 内訳）
7. Cloud Run 上で frontend / backend を単一 service にまとめるか、別 service に分けるか
8. DB migration を CI/CD パイプラインに含めるか、手動承認 step に切り出すか

---

## 推奨する最初の実装順

1. Phase 0 の土台
2. Phase 1 の inventory / reservation / shortage
3. Phase 2 の procurement 内部モデル
4. Phase 3 の OCR
5. Phase 4 の adapter / webhook / reconciliation
6. Phase 5 の auth 骨組み
7. Phase 6 の画面統合
8. Phase 7 の同期運用補強
9. Phase 8 の CI/CD・Cloud Run デプロイ整備

---

## CI/CD と Cloud Run の前提整理

実装計画の終点を Cloud Run deploy まで伸ばすなら、最低限以下を別紙ではなく本計画の管理対象として持つ。

### GitHub Actions で必要になる主な repository variables / secrets

* `GCP_PROJECT_ID`
* `GCP_PROJECT_NUMBER`
* `GCP_REGION`
* `GAR_LOCATION`
* `GAR_REPOSITORY`
* `CLOUD_RUN_SERVICE`
* `CLOUD_RUN_SERVICE_ACCOUNT`
* `WORKLOAD_IDENTITY_PROVIDER`
* `WORKLOAD_IDENTITY_SERVICE_ACCOUNT`
* `DEPLOY_ENV`
* `APP_BASE_URL`
* `API_BASE_URL`
* `DATABASE_URL`
* `APP_ENV`
* `AUTH_MODE`
* `RBAC_MODE`
* `STORAGE_MODE`
* `GOOGLE_CLOUD_PROJECT`
* `GCS_BUCKET_ARTIFACTS` または同等の artifact 保存先
* `VERTEX_AI_LOCATION`
* `GEMINI_MODEL`
* `JWT_ISSUER` / `JWT_AUDIENCE` / `JWKS_URL`（Auth を Cloud mode で使う場合）
* `RAKURAKU_API_BASE_URL`
* `RAKURAKU_API_TOKEN`
* `BAKURAKU_API_BASE_URL`
* `BAKURAKU_API_TOKEN`
* `WEBHOOK_SHARED_SECRET` または署名検証に必要な値

### GCP 側で事前に必要になること

* Cloud Run API 有効化
* Artifact Registry API 有効化
* IAM / Service Account Credentials / STS 系 API 有効化
* deploy 用 service account 作成
* GitHub OIDC 連携用 Workload Identity Pool / Provider 作成
* Artifact Registry repository 作成
* Cloud Run service 用 runtime service account 作成
* PostgreSQL 接続先の確保
  * Cloud SQL を使うなら接続方式と権限
  * 外部 PostgreSQL を使うなら接続元制御
* Cloud Storage bucket 作成
* Secret Manager を使うなら secret 登録と参照権限付与

### deploy 成立の判断基準

* GitHub 上の `main` push を契機に workflow が自動起動する
* workflow が image build / push / deploy を完了する
* Cloud Run revision が Ready になる
* `/health` または `/ready` が成功を返す
* アプリの必須 env 欠落時に fail fast する
* rollback に必要な revision / image tag が追跡できる

---

## 補足

仕様書中の `GUI` 表記は backend + frontend をまとめた表現として解釈する。
ただし、実装上は以下に統一する。

* 外部通信: backend
* webhook 受信: backend
* 外部再取得: backend
* UI 表示: frontend は backend API のみ参照

