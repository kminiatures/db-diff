package database

import (
	"database/sql"
	"fmt"
	"strings"

	_ "github.com/lib/pq"
	"github.com/koba/db-diff/internal/schema"
)

// Postgres implements the Database interface for PostgreSQL
type Postgres struct {
	config Config
	db     *sql.DB
}

// NewPostgres creates a new PostgreSQL database connection
func NewPostgres(config Config) *Postgres {
	return &Postgres{config: config}
}

// Connect establishes a connection to PostgreSQL
func (p *Postgres) Connect() error {
	dsn := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		p.config.Host,
		p.config.Port,
		p.config.User,
		p.config.Password,
		p.config.Database,
	)

	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return fmt.Errorf("failed to open PostgreSQL connection: %w", err)
	}

	if err := db.Ping(); err != nil {
		return fmt.Errorf("failed to ping PostgreSQL: %w", err)
	}

	p.db = db
	return nil
}

// Close closes the PostgreSQL connection
func (p *Postgres) Close() error {
	if p.db != nil {
		return p.db.Close()
	}
	return nil
}

// GetAllTables retrieves all table names in the public schema
func (p *Postgres) GetAllTables() ([]string, error) {
	query := `
		SELECT table_name
		FROM information_schema.tables
		WHERE table_schema = 'public' AND table_type = 'BASE TABLE'
		ORDER BY table_name
	`
	rows, err := p.db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to get tables: %w", err)
	}
	defer rows.Close()

	var tables []string
	for rows.Next() {
		var tableName string
		if err := rows.Scan(&tableName); err != nil {
			return nil, fmt.Errorf("failed to scan table name: %w", err)
		}
		tables = append(tables, tableName)
	}

	return tables, rows.Err()
}

// GetTableSchema retrieves the schema for a specific table
func (p *Postgres) GetTableSchema(tableName string) (*schema.TableSchema, error) {
	tableSchema := &schema.TableSchema{
		Name:        tableName,
		Columns:     []schema.Column{},
		Indexes:     []schema.Index{},
		ForeignKeys: []schema.ForeignKey{},
	}

	// Get columns
	columns, err := p.getColumns(tableName)
	if err != nil {
		return nil, err
	}
	tableSchema.Columns = columns

	// Get indexes
	indexes, err := p.getIndexes(tableName)
	if err != nil {
		return nil, err
	}
	tableSchema.Indexes = indexes

	// Get foreign keys
	foreignKeys, err := p.getForeignKeys(tableName)
	if err != nil {
		return nil, err
	}
	tableSchema.ForeignKeys = foreignKeys

	return tableSchema, nil
}

func (p *Postgres) getColumns(tableName string) ([]schema.Column, error) {
	query := `
		SELECT
			column_name,
			data_type,
			is_nullable,
			column_default,
			ordinal_position
		FROM information_schema.columns
		WHERE table_schema = 'public' AND table_name = $1
		ORDER BY ordinal_position
	`
	rows, err := p.db.Query(query, tableName)
	if err != nil {
		return nil, fmt.Errorf("failed to get columns: %w", err)
	}
	defer rows.Close()

	var columns []schema.Column
	for rows.Next() {
		var col schema.Column
		var nullable string
		var defaultValue sql.NullString

		if err := rows.Scan(&col.Name, &col.Type, &nullable, &defaultValue, &col.Position); err != nil {
			return nil, fmt.Errorf("failed to scan column: %w", err)
		}

		col.Nullable = (nullable == "YES")
		if defaultValue.Valid {
			col.DefaultValue = &defaultValue.String
		}

		// Check for serial/identity columns (auto increment)
		if strings.Contains(strings.ToLower(defaultValue.String), "nextval") {
			col.AutoIncrement = true
		}

		columns = append(columns, col)
	}

	return columns, rows.Err()
}

func (p *Postgres) getIndexes(tableName string) ([]schema.Index, error) {
	query := `
		SELECT
			i.relname AS index_name,
			a.attname AS column_name,
			ix.indisunique AS is_unique,
			ix.indisprimary AS is_primary
		FROM pg_class t
		JOIN pg_index ix ON t.oid = ix.indrelid
		JOIN pg_class i ON i.oid = ix.indexrelid
		JOIN pg_attribute a ON a.attrelid = t.oid AND a.attnum = ANY(ix.indkey)
		WHERE t.relname = $1 AND t.relkind = 'r'
		ORDER BY i.relname, a.attnum
	`
	rows, err := p.db.Query(query, tableName)
	if err != nil {
		return nil, fmt.Errorf("failed to get indexes: %w", err)
	}
	defer rows.Close()

	indexMap := make(map[string]*schema.Index)
	for rows.Next() {
		var indexName, columnName string
		var isUnique, isPrimary bool

		if err := rows.Scan(&indexName, &columnName, &isUnique, &isPrimary); err != nil {
			return nil, fmt.Errorf("failed to scan index: %w", err)
		}

		if idx, exists := indexMap[indexName]; exists {
			idx.Columns = append(idx.Columns, columnName)
		} else {
			indexMap[indexName] = &schema.Index{
				Name:    indexName,
				Columns: []string{columnName},
				Unique:  isUnique,
				Primary: isPrimary,
				Type:    "BTREE", // PostgreSQL default
			}
		}
	}

	var indexes []schema.Index
	for _, idx := range indexMap {
		indexes = append(indexes, *idx)
	}

	return indexes, rows.Err()
}

func (p *Postgres) getForeignKeys(tableName string) ([]schema.ForeignKey, error) {
	query := `
		SELECT
			tc.constraint_name,
			kcu.column_name,
			ccu.table_name AS referenced_table,
			ccu.column_name AS referenced_column,
			rc.update_rule,
			rc.delete_rule
		FROM information_schema.table_constraints tc
		JOIN information_schema.key_column_usage kcu
			ON tc.constraint_name = kcu.constraint_name
			AND tc.table_schema = kcu.table_schema
		JOIN information_schema.constraint_column_usage ccu
			ON ccu.constraint_name = tc.constraint_name
			AND ccu.table_schema = tc.table_schema
		JOIN information_schema.referential_constraints rc
			ON rc.constraint_name = tc.constraint_name
		WHERE tc.constraint_type = 'FOREIGN KEY'
			AND tc.table_schema = 'public'
			AND tc.table_name = $1
	`
	rows, err := p.db.Query(query, tableName)
	if err != nil {
		return nil, fmt.Errorf("failed to get foreign keys: %w", err)
	}
	defer rows.Close()

	var foreignKeys []schema.ForeignKey
	for rows.Next() {
		var fk schema.ForeignKey

		if err := rows.Scan(&fk.Name, &fk.Column, &fk.ReferencedTable, &fk.ReferencedColumn, &fk.OnUpdate, &fk.OnDelete); err != nil {
			return nil, fmt.Errorf("failed to scan foreign key: %w", err)
		}

		foreignKeys = append(foreignKeys, fk)
	}

	return foreignKeys, rows.Err()
}

// GetTableData retrieves all data from a table
func (p *Postgres) GetTableData(tableName string, limit int) ([]schema.Row, error) {
	query := fmt.Sprintf("SELECT * FROM \"%s\"", tableName)
	if limit > 0 {
		query = fmt.Sprintf("%s LIMIT %d", query, limit)
	}

	rows, err := p.db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to get table data: %w", err)
	}
	defer rows.Close()

	columns, err := rows.Columns()
	if err != nil {
		return nil, fmt.Errorf("failed to get columns: %w", err)
	}

	var data []schema.Row
	for rows.Next() {
		values := make([]interface{}, len(columns))
		valuePtrs := make([]interface{}, len(columns))
		for i := range values {
			valuePtrs[i] = &values[i]
		}

		if err := rows.Scan(valuePtrs...); err != nil {
			return nil, fmt.Errorf("failed to scan row: %w", err)
		}

		row := make(schema.Row)
		for i, col := range columns {
			val := values[i]
			if b, ok := val.([]byte); ok {
				row[col] = string(b)
			} else {
				row[col] = val
			}
		}

		data = append(data, row)
	}

	return data, rows.Err()
}
