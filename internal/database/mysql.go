package database

import (
	"database/sql"
	"fmt"
	"strings"

	_ "github.com/go-sql-driver/mysql"
	"github.com/koba/db-diff/internal/schema"
)

// MySQL implements the Database interface for MySQL
type MySQL struct {
	config Config
	db     *sql.DB
}

// NewMySQL creates a new MySQL database connection
func NewMySQL(config Config) *MySQL {
	return &MySQL{config: config}
}

// Connect establishes a connection to MySQL
func (m *MySQL) Connect() error {
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?parseTime=true",
		m.config.User,
		m.config.Password,
		m.config.Host,
		m.config.Port,
		m.config.Database,
	)

	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return fmt.Errorf("failed to open MySQL connection: %w", err)
	}

	if err := db.Ping(); err != nil {
		return fmt.Errorf("failed to ping MySQL: %w", err)
	}

	m.db = db
	return nil
}

// Close closes the MySQL connection
func (m *MySQL) Close() error {
	if m.db != nil {
		return m.db.Close()
	}
	return nil
}

// GetAllTables retrieves all table names in the database
func (m *MySQL) GetAllTables() ([]string, error) {
	query := "SELECT TABLE_NAME FROM information_schema.TABLES WHERE TABLE_SCHEMA = ? ORDER BY TABLE_NAME"
	rows, err := m.db.Query(query, m.config.Database)
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
func (m *MySQL) GetTableSchema(tableName string) (*schema.TableSchema, error) {
	tableSchema := &schema.TableSchema{
		Name:    tableName,
		Columns: []schema.Column{},
		Indexes: []schema.Index{},
		ForeignKeys: []schema.ForeignKey{},
	}

	// Get columns
	columns, err := m.getColumns(tableName)
	if err != nil {
		return nil, err
	}
	tableSchema.Columns = columns

	// Get indexes
	indexes, err := m.getIndexes(tableName)
	if err != nil {
		return nil, err
	}
	tableSchema.Indexes = indexes

	// Get foreign keys
	foreignKeys, err := m.getForeignKeys(tableName)
	if err != nil {
		return nil, err
	}
	tableSchema.ForeignKeys = foreignKeys

	return tableSchema, nil
}

func (m *MySQL) getColumns(tableName string) ([]schema.Column, error) {
	query := `
		SELECT
			COLUMN_NAME,
			COLUMN_TYPE,
			IS_NULLABLE,
			COLUMN_DEFAULT,
			EXTRA,
			ORDINAL_POSITION
		FROM information_schema.COLUMNS
		WHERE TABLE_SCHEMA = ? AND TABLE_NAME = ?
		ORDER BY ORDINAL_POSITION
	`
	rows, err := m.db.Query(query, m.config.Database, tableName)
	if err != nil {
		return nil, fmt.Errorf("failed to get columns: %w", err)
	}
	defer rows.Close()

	var columns []schema.Column
	for rows.Next() {
		var col schema.Column
		var nullable string
		var defaultValue sql.NullString
		var extra string

		if err := rows.Scan(&col.Name, &col.Type, &nullable, &defaultValue, &extra, &col.Position); err != nil {
			return nil, fmt.Errorf("failed to scan column: %w", err)
		}

		col.Nullable = (nullable == "YES")
		if defaultValue.Valid {
			col.DefaultValue = &defaultValue.String
		}
		col.AutoIncrement = strings.Contains(strings.ToLower(extra), "auto_increment")

		columns = append(columns, col)
	}

	return columns, rows.Err()
}

func (m *MySQL) getIndexes(tableName string) ([]schema.Index, error) {
	query := `
		SELECT
			INDEX_NAME,
			COLUMN_NAME,
			NON_UNIQUE,
			INDEX_TYPE
		FROM information_schema.STATISTICS
		WHERE TABLE_SCHEMA = ? AND TABLE_NAME = ?
		ORDER BY INDEX_NAME, SEQ_IN_INDEX
	`
	rows, err := m.db.Query(query, m.config.Database, tableName)
	if err != nil {
		return nil, fmt.Errorf("failed to get indexes: %w", err)
	}
	defer rows.Close()

	indexMap := make(map[string]*schema.Index)
	for rows.Next() {
		var indexName, columnName, indexType string
		var nonUnique int

		if err := rows.Scan(&indexName, &columnName, &nonUnique, &indexType); err != nil {
			return nil, fmt.Errorf("failed to scan index: %w", err)
		}

		if idx, exists := indexMap[indexName]; exists {
			idx.Columns = append(idx.Columns, columnName)
		} else {
			indexMap[indexName] = &schema.Index{
				Name:    indexName,
				Columns: []string{columnName},
				Unique:  nonUnique == 0,
				Primary: indexName == "PRIMARY",
				Type:    indexType,
			}
		}
	}

	var indexes []schema.Index
	for _, idx := range indexMap {
		indexes = append(indexes, *idx)
	}

	return indexes, rows.Err()
}

func (m *MySQL) getForeignKeys(tableName string) ([]schema.ForeignKey, error) {
	query := `
		SELECT
			CONSTRAINT_NAME,
			COLUMN_NAME,
			REFERENCED_TABLE_NAME,
			REFERENCED_COLUMN_NAME
		FROM information_schema.KEY_COLUMN_USAGE
		WHERE TABLE_SCHEMA = ? AND TABLE_NAME = ? AND REFERENCED_TABLE_NAME IS NOT NULL
	`
	rows, err := m.db.Query(query, m.config.Database, tableName)
	if err != nil {
		return nil, fmt.Errorf("failed to get foreign keys: %w", err)
	}
	defer rows.Close()

	var foreignKeys []schema.ForeignKey
	for rows.Next() {
		var fk schema.ForeignKey

		if err := rows.Scan(&fk.Name, &fk.Column, &fk.ReferencedTable, &fk.ReferencedColumn); err != nil {
			return nil, fmt.Errorf("failed to scan foreign key: %w", err)
		}

		// Get ON DELETE and ON UPDATE actions
		actionQuery := `
			SELECT DELETE_RULE, UPDATE_RULE
			FROM information_schema.REFERENTIAL_CONSTRAINTS
			WHERE CONSTRAINT_SCHEMA = ? AND CONSTRAINT_NAME = ?
		`
		err := m.db.QueryRow(actionQuery, m.config.Database, fk.Name).Scan(&fk.OnDelete, &fk.OnUpdate)
		if err != nil {
			return nil, fmt.Errorf("failed to get FK actions: %w", err)
		}

		foreignKeys = append(foreignKeys, fk)
	}

	return foreignKeys, rows.Err()
}

// GetTableData retrieves all data from a table
func (m *MySQL) GetTableData(tableName string, limit int) ([]schema.Row, error) {
	query := fmt.Sprintf("SELECT * FROM `%s`", tableName)
	if limit > 0 {
		query = fmt.Sprintf("%s LIMIT %d", query, limit)
	}

	rows, err := m.db.Query(query)
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
