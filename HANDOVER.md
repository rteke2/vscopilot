# 引き継ぎドキュメント

更新日: 2026-04-27
対象リポジトリ: vscopilot

## 1. プロジェクト目的

PC上のVS Code Copilot Chatを、スマホ経由でトリガーして内容を取得し、Google Apps Script (GAS) + スプレッドシートで保存・表示する。

最終的には、GAS側から入力したメッセージをVS Code Copilot Chatへ送信できる双方向化を目指す。

## 2. これまでの指示内容（要約）

- スマホからVS CodeのCopilot Chatをコントロールしたい。
- Chat内容を取得してAPIへ送る。
- Go言語リスナーで受け、トリガーにする。
- トリガー時にVS Codeプロセスを確認し、Chat内容を取得する。
- 取得内容をGAS APIへ送る。
- GASはスプレッドシートへ保存する。
- GAS表示画面でスプレッドシート内容を表示し、更新トリガー送信ボタンを置く。
- 次段階として、GASフォーム入力をVS Code Chatへ送信する機能を追加する。

## 3. 現在の実装状況

MVPとして、以下を実装済み。

- Goリスナー
  - /health と /trigger を提供
  - トリガー時にVS Codeプロセス検出
  - Copilot Chatログ探索・最新内容抽出
  - GAS Web AppへJSON転送
- Copilotログ読み取り
  - workspaceStorage配下の GitHub.copilot-chat/debug-logs を探索
  - 最新ファイルを選び、user/assistantの最新メッセージを抽出
  - COPILOT_LOG_ROOTS で探索ルート上書き可能
- GAS
  - doPostでログ保存
  - action=trigger でGoへトリガー転送
  - doGetで一覧画面を表示
  - シート未作成時に自動作成
- README
  - セットアップ手順、環境変数、エンドポイント、次段階方針を記載

## 4. 主要ファイル

- cmd/listener/main.go
- internal/copilot/reader.go
- internal/bridge/types.go
- gas/Code.gs
- gas/Index.html
- README.md

## 5. 動作フロー（MVP）

1. スマホでGAS Web Appを開く
2. 「更新トリガー送信」を押す
3. GASがGoの /trigger を呼び出す
4. GoがCopilot Chatログを取得
5. GoがGAS Web Appへ保存用POST
6. GASがスプレッドシートへappend
7. GAS画面に最新履歴を表示

## 6. 必須設定

### Go側環境変数

- GAS_WEBHOOK_URL: GAS Web App URL
- TRIGGER_TOKEN: 共有トークン（任意）
- LISTEN_ADDR: 例 :8080（任意）
- COPILOT_LOG_ROOTS: 探索ルート上書き（任意）

### GAS側Script Properties

- GO_TRIGGER_URL: Goの /trigger URL
- TRIGGER_TOKEN: Go側と合わせる

## 7. 検証メモ

- 直近のビルド結果: go build ./... は成功
- VS Code拡張の内部ログ形式は将来変更の可能性があるため、reader.go は追加調整が必要になる場合がある

## 8. 既知の制約

- Go実行環境からVS Codeログディレクトリにアクセスできることが前提
- スマホからGo到達のため、公開経路（トンネル等）が必要
- 現時点では「取得・保存・表示・トリガー送信」の一方向連携のみ

## 9. 次フェーズ（実装予定）

GASフォーム入力 → Go受信 → VS Code Copilot Chatへ投入。

候補:

- VS Code拡張でローカル受信APIを作り、コマンド実行でチャット投入
- もしくは入力キュー方式（Spreadsheet/Firestore）でVS Code側コンパニオンがポーリング

## 10. 引き継ぎ時の優先タスク

1. E2E疎通確認（スマホ操作からシート反映まで）
2. エラーハンドリング強化（GAS/Go双方の異常時メッセージ）
3. Chat抽出精度向上（ログ形式差分への耐性）
4. 次フェーズの送信方式決定（拡張方式 or キュー方式）
