# db-diff

データベースのテーブル構造とデータのスナップショットを取得し、差分を比較・SQL生成するCLIツール

## 特徴

- **スナップショット取得**: MySQL/PostgreSQLのテーブル構造とデータをSQLite形式で保存
- **差分比較**: 2つのスナップショット間のスキーマとデータの違いを表示
- **SQL生成**: 差分を解消するDDL/DMLを自動生成

## インストール

```bash
go install github.com/koba/db-diff/cmd/dbdiff@latest
```

または、ソースからビルド:

```bash
git clone https://github.com/koba/db-diff.git
cd db-diff
go build -o dbdiff ./cmd/dbdiff
```

## 環境変数

以下の環境変数を設定してデータベースに接続します:

```bash
export DB_TYPE=mysql        # または postgres
export DB_HOST=localhost
export DB_PORT=3306         # デフォルト: MySQL=3306, PostgreSQL=5432
export DB_NAME=mydb
export DB_USER=root
export DB_PASSWORD=password
```

## 使い方

### 1. スナップショット作成

```bash
# 全テーブルのスナップショットを作成
dbdiff snapshot

# スナップショット名を指定
dbdiff snapshot mydb-before-migration

# 特定のテーブルのみ
dbdiff snapshot --tables users,posts,comments

# 行数を制限
dbdiff snapshot --limit 1000

# 保存先を指定
dbdiff snapshot --output-dir /path/to/snapshots
```

スナップショットは `./snapshots/` ディレクトリに保存されます（デフォルト）。

### 2. 差分比較

```bash
# 2つのスナップショットを比較
dbdiff diff snapshots/mydb-2026-02-07-10-00-00.db snapshots/mydb-2026-02-07-11-00-00.db
```

出力例:
```
=== Schema Differences ===

Table: users
  Action: MODIFY
  Column changes:
    - email: ADD
    - age: MODIFY

=== Data Differences ===

Table: users
  Rows added: 5
  Rows deleted: 2
  Rows modified: 10
```

### 3. マイグレーションSQL生成

```bash
# 差分を解消するSQLを生成
dbdiff migrate snapshots/snapshot1.db snapshots/snapshot2.db
```

出力例:
```sql
-- Migration SQL from snapshot1.db to snapshot2.db
-- Generated at: 2026-02-07T14:00:00+09:00

ALTER TABLE `users` ADD COLUMN `email` varchar(255) NOT NULL;
ALTER TABLE `users` MODIFY COLUMN `age` int;

DELETE FROM `users` WHERE `id` = 1;
INSERT INTO `users` (`id`, `name`, `email`) VALUES (100, 'John', 'john@example.com');
UPDATE `users` SET `email` = 'new@example.com' WHERE `id` = 50;
```

## プロジェクト構造

```
db-diff/
├── cmd/dbdiff/          # CLIエントリーポイント
├── internal/
│   ├── database/        # DB接続層（MySQL/PostgreSQL）
│   ├── schema/          # スキーマ定義
│   ├── snapshot/        # スナップショット作成・読込
│   ├── diff/            # 差分比較
│   └── generator/       # DDL/DML生成
└── snapshots/           # スナップショット保存先（.gitignore）
```

## 開発

### ビルド

```bash
make build
# または
go build -o dbdiff ./cmd/dbdiff
```

### テスト

Docker Composeを使ってMySQLとPostgreSQLのテスト環境を起動できます：

```bash
# MySQLの完全なテストシナリオを実行
make test-mysql

# PostgreSQLの完全なテストシナリオを実行
make test-postgres

# 手動でテストする場合
make docker-up         # コンテナ起動
make build             # ビルド
# test/README.md の手順に従ってテスト
make docker-down       # コンテナ停止
```

詳細なテスト手順は [test/README.md](test/README.md) を参照してください。

### その他のコマンド

```bash
make help              # 利用可能なコマンドを表示
make lint              # コードをリント
make clean             # ビルド成果物を削除
make deps              # 依存関係を更新
make docker-reset      # Docker環境を初期化
```

## プロジェクト構成

```
db-diff/
├── cmd/dbdiff/          # CLIエントリーポイント
├── internal/
│   ├── database/        # DB接続層（MySQL/PostgreSQL）
│   ├── schema/          # スキーマ定義
│   ├── snapshot/        # スナップショット作成・読込
│   ├── diff/            # 差分比較
│   └── generator/       # DDL/DML生成
├── test/                # テスト用データとスクリプト
│   ├── mysql/           # MySQL初期化スクリプト
│   └── postgres/        # PostgreSQL初期化スクリプト
├── docker-compose.yml   # テスト用Docker環境
├── Makefile             # ビルド・テストコマンド
└── snapshots/           # スナップショット保存先（.gitignore）
```

## ライセンス

MIT License
