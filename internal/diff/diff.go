package diff

import (
	"fmt"

	"github.com/koba/db-diff/internal/snapshot"
)

// DiffResult holds the complete comparison result
type DiffResult struct {
	SchemaDiffs map[string]*SchemaDiff
	DataDiffs   map[string]*DataDiff
}

// Compare compares two snapshots and returns the differences
func Compare(snap1, snap2 *snapshot.Snapshot) *DiffResult {
	result := &DiffResult{
		SchemaDiffs: make(map[string]*SchemaDiff),
		DataDiffs:   make(map[string]*DataDiff),
	}

	// Find all unique table names
	tableNames := make(map[string]bool)
	for name := range snap1.Tables {
		tableNames[name] = true
	}
	for name := range snap2.Tables {
		tableNames[name] = true
	}

	// Compare each table
	for tableName := range tableNames {
		table1, exists1 := snap1.Tables[tableName]
		table2, exists2 := snap2.Tables[tableName]

		if !exists1 {
			// Table added in snapshot2
			result.SchemaDiffs[tableName] = &SchemaDiff{
				TableName: tableName,
				Action:    ActionAdd,
				NewSchema: &table2.Schema,
			}
			continue
		}

		if !exists2 {
			// Table removed in snapshot2
			result.SchemaDiffs[tableName] = &SchemaDiff{
				TableName: tableName,
				Action:    ActionDrop,
				OldSchema: &table1.Schema,
			}
			continue
		}

		// Table exists in both snapshots - compare schema
		schemaDiff := compareSchemas(&table1.Schema, &table2.Schema)
		if schemaDiff != nil {
			result.SchemaDiffs[tableName] = schemaDiff
		}

		// Compare data
		dataDiff := compareData(tableName, table1.Data, table2.Data, &table2.Schema)
		if dataDiff != nil {
			result.DataDiffs[tableName] = dataDiff
		}
	}

	return result
}

// Display prints the diff result in a human-readable format
func Display(result *DiffResult) {
	if len(result.SchemaDiffs) == 0 && len(result.DataDiffs) == 0 {
		fmt.Println("No differences found.")
		return
	}

	// Display schema differences
	if len(result.SchemaDiffs) > 0 {
		fmt.Println("=== Schema Differences ===")
		fmt.Println()
		for tableName, schemaDiff := range result.SchemaDiffs {
			displaySchemaDiff(tableName, schemaDiff)
		}
	}

	// Display data differences
	if len(result.DataDiffs) > 0 {
		fmt.Println()
		fmt.Println("=== Data Differences ===")
		fmt.Println()
		for tableName, dataDiff := range result.DataDiffs {
			displayDataDiff(tableName, dataDiff)
		}
	}
}

func displaySchemaDiff(tableName string, diff *SchemaDiff) {
	fmt.Printf("Table: %s\n", tableName)

	switch diff.Action {
	case ActionAdd:
		fmt.Printf("  Action: ADD (new table)\n")
		fmt.Printf("  Columns: %d\n", len(diff.NewSchema.Columns))
	case ActionDrop:
		fmt.Printf("  Action: DROP (removed table)\n")
	case ActionModify:
		fmt.Printf("  Action: MODIFY\n")
		if len(diff.ColumnChanges) > 0 {
			fmt.Printf("  Column changes:\n")
			for _, change := range diff.ColumnChanges {
				fmt.Printf("    - %s: %s\n", change.ColumnName, change.Action)
			}
		}
		if len(diff.IndexChanges) > 0 {
			fmt.Printf("  Index changes:\n")
			for _, change := range diff.IndexChanges {
				fmt.Printf("    - %s: %s\n", change.IndexName, change.Action)
			}
		}
		if len(diff.ForeignKeyChanges) > 0 {
			fmt.Printf("  Foreign key changes:\n")
			for _, change := range diff.ForeignKeyChanges {
				fmt.Printf("    - %s: %s\n", change.FKName, change.Action)
			}
		}
	}
	fmt.Println()
}

func displayDataDiff(tableName string, diff *DataDiff) {
	fmt.Printf("Table: %s\n", tableName)
	fmt.Printf("  Rows added: %d\n", len(diff.RowsAdded))
	fmt.Printf("  Rows deleted: %d\n", len(diff.RowsDeleted))
	fmt.Printf("  Rows modified: %d\n", len(diff.RowsModified))
	fmt.Println()
}
