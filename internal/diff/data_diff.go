package diff

import (
	"encoding/json"
	"fmt"

	"github.com/koba/db-diff/internal/schema"
)

// DataDiff represents data differences for a table
type DataDiff struct {
	TableName    string
	RowsAdded    []schema.Row
	RowsDeleted  []schema.Row
	RowsModified []RowModification
}

// RowModification represents a modified row
type RowModification struct {
	OldRow schema.Row
	NewRow schema.Row
}

// compareData compares data between two tables
func compareData(tableName string, oldData, newData []schema.Row, tableSchema *schema.TableSchema) *DataDiff {
	diff := &DataDiff{
		TableName:    tableName,
		RowsAdded:    []schema.Row{},
		RowsDeleted:  []schema.Row{},
		RowsModified: []RowModification{},
	}

	// Find primary key columns
	pkColumns := getPrimaryKeyColumns(tableSchema)
	if len(pkColumns) == 0 {
		// No primary key - cannot reliably compare data
		// Fall back to treating all rows as different
		if len(oldData) != len(newData) {
			diff.RowsDeleted = oldData
			diff.RowsAdded = newData
		}
		return diff
	}

	// Create maps keyed by primary key
	oldRows := make(map[string]schema.Row)
	for _, row := range oldData {
		key := rowKey(row, pkColumns)
		oldRows[key] = row
	}

	newRows := make(map[string]schema.Row)
	for _, row := range newData {
		key := rowKey(row, pkColumns)
		newRows[key] = row
	}

	// Find added and modified rows
	for key, newRow := range newRows {
		if oldRow, exists := oldRows[key]; exists {
			if !rowsEqual(oldRow, newRow) {
				diff.RowsModified = append(diff.RowsModified, RowModification{
					OldRow: oldRow,
					NewRow: newRow,
				})
			}
		} else {
			diff.RowsAdded = append(diff.RowsAdded, newRow)
		}
	}

	// Find deleted rows
	for key, oldRow := range oldRows {
		if _, exists := newRows[key]; !exists {
			diff.RowsDeleted = append(diff.RowsDeleted, oldRow)
		}
	}

	// Return nil if no changes
	if len(diff.RowsAdded) == 0 && len(diff.RowsDeleted) == 0 && len(diff.RowsModified) == 0 {
		return nil
	}

	return diff
}

// getPrimaryKeyColumns returns the primary key column names
func getPrimaryKeyColumns(tableSchema *schema.TableSchema) []string {
	var pkColumns []string

	for _, index := range tableSchema.Indexes {
		if index.Primary {
			pkColumns = index.Columns
			break
		}
	}

	return pkColumns
}

// rowKey generates a unique key for a row based on primary key columns
func rowKey(row schema.Row, pkColumns []string) string {
	keyParts := make([]interface{}, len(pkColumns))
	for i, col := range pkColumns {
		keyParts[i] = row[col]
	}

	// Use JSON encoding for consistent key generation
	keyJSON, err := json.Marshal(keyParts)
	if err != nil {
		return fmt.Sprintf("%v", keyParts)
	}

	return string(keyJSON)
}

// rowsEqual checks if two rows are equal
func rowsEqual(a, b schema.Row) bool {
	if len(a) != len(b) {
		return false
	}

	for key, valA := range a {
		valB, exists := b[key]
		if !exists {
			return false
		}

		// Use JSON comparison for consistent equality check
		jsonA, _ := json.Marshal(valA)
		jsonB, _ := json.Marshal(valB)

		if string(jsonA) != string(jsonB) {
			return false
		}
	}

	return true
}
