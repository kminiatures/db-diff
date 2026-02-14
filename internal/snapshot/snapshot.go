package snapshot

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	_ "modernc.org/sqlite"

	"github.com/koba/db-diff/internal/database"
	"github.com/koba/db-diff/internal/schema"
)

// Snapshot represents a database snapshot
type Snapshot struct {
	Metadata map[string]string
	Tables   map[string]*schema.Table
}

// CreateSnapshot creates a snapshot of the database
func CreateSnapshot(db database.Database, tables []string, outputPath string, limit int) error {
	// Ensure output directory exists
	dir := filepath.Dir(outputPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	// Remove existing snapshot file if it exists
	if _, err := os.Stat(outputPath); err == nil {
		if err := os.Remove(outputPath); err != nil {
			return fmt.Errorf("failed to remove existing snapshot: %w", err)
		}
	}

	// Create SQLite database
	snapshotDB, err := sql.Open("sqlite", outputPath)
	if err != nil {
		return fmt.Errorf("failed to create snapshot database: %w", err)
	}
	defer snapshotDB.Close()

	// Initialize schema
	if err := initializeSchema(snapshotDB); err != nil {
		return fmt.Errorf("failed to initialize snapshot schema: %w", err)
	}

	// Store metadata
	metadata := map[string]string{
		"created_at": time.Now().Format(time.RFC3339),
		"db_type":    "unknown", // Could be enhanced to detect DB type
	}

	for key, value := range metadata {
		_, err := snapshotDB.Exec("INSERT INTO metadata (key, value) VALUES (?, ?)", key, value)
		if err != nil {
			return fmt.Errorf("failed to insert metadata: %w", err)
		}
	}

	// Get all tables if not specified
	if len(tables) == 0 {
		tables, err = db.GetAllTables()
		if err != nil {
			return fmt.Errorf("failed to get all tables: %w", err)
		}
	}

	// Snapshot each table
	for _, tableName := range tables {
		if err := snapshotTable(db, snapshotDB, tableName, limit); err != nil {
			return fmt.Errorf("failed to snapshot table %s: %w", tableName, err)
		}
	}

	return nil
}

func snapshotTable(db database.Database, snapshotDB *sql.DB, tableName string, limit int) error {
	// Get table schema
	tableSchema, err := db.GetTableSchema(tableName)
	if err != nil {
		return fmt.Errorf("failed to get schema: %w", err)
	}

	// Store schema as JSON
	schemaJSON, err := json.Marshal(tableSchema)
	if err != nil {
		return fmt.Errorf("failed to marshal schema: %w", err)
	}

	_, err = snapshotDB.Exec(
		"INSERT INTO table_schemas (table_name, schema_json) VALUES (?, ?)",
		tableName,
		string(schemaJSON),
	)
	if err != nil {
		return fmt.Errorf("failed to insert schema: %w", err)
	}

	// Get table data
	data, err := db.GetTableData(tableName, limit)
	if err != nil {
		return fmt.Errorf("failed to get data: %w", err)
	}

	// Store data as JSON
	tx, err := snapshotDB.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	stmt, err := tx.Prepare("INSERT INTO table_data (table_name, row_json) VALUES (?, ?)")
	if err != nil {
		return fmt.Errorf("failed to prepare statement: %w", err)
	}
	defer stmt.Close()

	for _, row := range data {
		rowJSON, err := json.Marshal(row)
		if err != nil {
			return fmt.Errorf("failed to marshal row: %w", err)
		}

		_, err = stmt.Exec(tableName, string(rowJSON))
		if err != nil {
			return fmt.Errorf("failed to insert row: %w", err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

// LoadSnapshot loads a snapshot from a SQLite file
func LoadSnapshot(snapshotPath string) (*Snapshot, error) {
	// Check if file exists
	if _, err := os.Stat(snapshotPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("snapshot file does not exist: %s", snapshotPath)
	}

	// Open SQLite database
	db, err := sql.Open("sqlite", snapshotPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open snapshot database: %w", err)
	}
	defer db.Close()

	snapshot := &Snapshot{
		Metadata: make(map[string]string),
		Tables:   make(map[string]*schema.Table),
	}

	// Load metadata
	rows, err := db.Query("SELECT key, value FROM metadata")
	if err != nil {
		return nil, fmt.Errorf("failed to query metadata: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var key, value string
		if err := rows.Scan(&key, &value); err != nil {
			return nil, fmt.Errorf("failed to scan metadata: %w", err)
		}
		snapshot.Metadata[key] = value
	}

	// Load table schemas
	schemaRows, err := db.Query("SELECT table_name, schema_json FROM table_schemas")
	if err != nil {
		return nil, fmt.Errorf("failed to query table schemas: %w", err)
	}
	defer schemaRows.Close()

	for schemaRows.Next() {
		var tableName, schemaJSON string
		if err := schemaRows.Scan(&tableName, &schemaJSON); err != nil {
			return nil, fmt.Errorf("failed to scan table schema: %w", err)
		}

		var tableSchema schema.TableSchema
		if err := json.Unmarshal([]byte(schemaJSON), &tableSchema); err != nil {
			return nil, fmt.Errorf("failed to unmarshal schema: %w", err)
		}

		snapshot.Tables[tableName] = &schema.Table{
			Schema: tableSchema,
			Data:   []schema.Row{},
		}
	}

	// Load table data
	for tableName := range snapshot.Tables {
		dataRows, err := db.Query("SELECT row_json FROM table_data WHERE table_name = ?", tableName)
		if err != nil {
			return nil, fmt.Errorf("failed to query table data: %w", err)
		}

		for dataRows.Next() {
			var rowJSON string
			if err := dataRows.Scan(&rowJSON); err != nil {
				dataRows.Close()
				return nil, fmt.Errorf("failed to scan row: %w", err)
			}

			var row schema.Row
			if err := json.Unmarshal([]byte(rowJSON), &row); err != nil {
				dataRows.Close()
				return nil, fmt.Errorf("failed to unmarshal row: %w", err)
			}

			snapshot.Tables[tableName].Data = append(snapshot.Tables[tableName].Data, row)
		}
		dataRows.Close()
	}

	return snapshot, nil
}
