package snapshot

import "database/sql"

const (
	// SQLite schema for storing snapshots
	createMetadataTable = `
		CREATE TABLE IF NOT EXISTS metadata (
			key TEXT PRIMARY KEY,
			value TEXT NOT NULL
		);
	`

	createTableSchemasTable = `
		CREATE TABLE IF NOT EXISTS table_schemas (
			table_name TEXT PRIMARY KEY,
			schema_json TEXT NOT NULL
		);
	`

	createTableDataTable = `
		CREATE TABLE IF NOT EXISTS table_data (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			table_name TEXT NOT NULL,
			row_json TEXT NOT NULL
		);
	`

	createTableDataIndex = `
		CREATE INDEX IF NOT EXISTS idx_table_data_table_name
		ON table_data(table_name);
	`
)

// InitializeSchema creates the necessary tables in the SQLite snapshot database
func initializeSchema(db *sql.DB) error {
	schemas := []string{
		createMetadataTable,
		createTableSchemasTable,
		createTableDataTable,
		createTableDataIndex,
	}

	for _, schema := range schemas {
		if _, err := db.Exec(schema); err != nil {
			return err
		}
	}

	return nil
}
