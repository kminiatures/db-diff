# テスト手順

このディレクトリには、db-diffツールをテストするためのDocker環境とサンプルデータが含まれています。

## 1. Docker環境の起動

```bash
# ルートディレクトリから実行
docker-compose up -d

# 起動確認
docker-compose ps
```

MySQLとPostgreSQLの両方が起動します：
- MySQL: localhost:3306
- PostgreSQL: localhost:5432

## 2. MySQLでのテスト

### 環境変数の設定

```bash
export DB_TYPE=mysql
export DB_HOST=localhost
export DB_PORT=3306
export DB_NAME=testdb
export DB_USER=testuser
export DB_PASSWORD=testpass
```

### スナップショット1の作成（初期状態）

```bash
go build -o dbdiff ./cmd/dbdiff
./dbdiff snapshot snapshot-before
```

`snapshots/snapshot-before.db` が作成されます。

### データベースを変更

```bash
# MySQLコンテナに接続してマイグレーションスクリプトを実行
docker exec -i dbdiff-mysql mysql -utestuser -ptestpass testdb < test/mysql/migration.sql
```

### スナップショット2の作成（変更後）

```bash
./dbdiff snapshot snapshot-after
```

### 差分の比較

```bash
./dbdiff diff snapshots/snapshot-before.db snapshots/snapshot-after.db
```

出力例:
```
=== Schema Differences ===

Table: users
  Action: MODIFY
  Column changes:
    - phone: ADD

Table: tags
  Action: ADD (new table)
  Columns: 3

=== Data Differences ===

Table: users
  Rows added: 1
  Rows deleted: 1
  Rows modified: 1

Table: posts
  Rows added: 1
  Rows modified: 1
```

### マイグレーションSQLの生成

```bash
./dbdiff migrate snapshots/snapshot-before.db snapshots/snapshot-after.db > migration.sql
```

生成されたSQLを確認:
```bash
cat migration.sql
```

## 3. PostgreSQLでのテスト

### 環境変数の設定

```bash
export DB_TYPE=postgres
export DB_HOST=localhost
export DB_PORT=5432
export DB_NAME=testdb
export DB_USER=testuser
export DB_PASSWORD=testpass
```

### スナップショット1の作成（初期状態）

```bash
./dbdiff snapshot pg-snapshot-before
```

### データベースを変更

```bash
# PostgreSQLコンテナに接続してマイグレーションスクリプトを実行
docker exec -i dbdiff-postgres psql -U testuser -d testdb < test/postgres/migration.sql
```

### スナップショット2の作成（変更後）

```bash
./dbdiff snapshot pg-snapshot-after
```

### 差分の比較

```bash
./dbdiff diff snapshots/pg-snapshot-before.db snapshots/pg-snapshot-after.db
```

### マイグレーションSQLの生成

```bash
./dbdiff migrate snapshots/pg-snapshot-before.db snapshots/pg-snapshot-after.db > pg-migration.sql
cat pg-migration.sql
```

## 4. その他の便利なコマンド

### MySQLに直接接続

```bash
docker exec -it dbdiff-mysql mysql -utestuser -ptestpass testdb
```

### PostgreSQLに直接接続

```bash
docker exec -it dbdiff-postgres psql -U testuser -d testdb
```

### 初期状態にリセット

```bash
docker-compose down -v
docker-compose up -d
```

### ログの確認

```bash
docker-compose logs mysql
docker-compose logs postgres
```

## 5. テストシナリオ

初期化スクリプトとマイグレーションスクリプトによって、以下の変更をテストできます：

### スキーマ変更
- ✓ 新しいカラムの追加 (users.phone)
- ✓ 新しいテーブルの追加 (tags)
- ✓ 新しいインデックスの追加 (posts.idx_published)

### データ変更
- ✓ 行の追加 (新ユーザー "dave")
- ✓ 行の削除 (ユーザー "charlie" とカスケード削除)
- ✓ 行の更新 (alice の email と age)
- ✓ 新しいポストの追加
- ✓ ポストのステータス更新

## 6. クリーンアップ

```bash
# コンテナとボリュームを削除
docker-compose down -v

# スナップショットファイルを削除
rm -rf snapshots/
```
