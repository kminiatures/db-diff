package diff

import (
	"github.com/koba/db-diff/internal/schema"
)

// Action represents the type of change
type Action string

const (
	ActionAdd    Action = "ADD"
	ActionDrop   Action = "DROP"
	ActionModify Action = "MODIFY"
)

// SchemaDiff represents schema differences for a table
type SchemaDiff struct {
	TableName         string
	Action            Action
	OldSchema         *schema.TableSchema
	NewSchema         *schema.TableSchema
	ColumnChanges     []ColumnChange
	IndexChanges      []IndexChange
	ForeignKeyChanges []ForeignKeyChange
}

// ColumnChange represents a change to a column
type ColumnChange struct {
	ColumnName string
	Action     Action
	OldColumn  *schema.Column
	NewColumn  *schema.Column
}

// IndexChange represents a change to an index
type IndexChange struct {
	IndexName string
	Action    Action
	OldIndex  *schema.Index
	NewIndex  *schema.Index
}

// ForeignKeyChange represents a change to a foreign key
type ForeignKeyChange struct {
	FKName       string
	Action       Action
	OldForeignKey *schema.ForeignKey
	NewForeignKey *schema.ForeignKey
}

// compareSchemas compares two table schemas
func compareSchemas(old, new *schema.TableSchema) *SchemaDiff {
	diff := &SchemaDiff{
		TableName:         new.Name,
		Action:            ActionModify,
		OldSchema:         old,
		NewSchema:         new,
		ColumnChanges:     []ColumnChange{},
		IndexChanges:      []IndexChange{},
		ForeignKeyChanges: []ForeignKeyChange{},
	}

	// Compare columns
	oldColumns := make(map[string]*schema.Column)
	for i := range old.Columns {
		oldColumns[old.Columns[i].Name] = &old.Columns[i]
	}

	newColumns := make(map[string]*schema.Column)
	for i := range new.Columns {
		newColumns[new.Columns[i].Name] = &new.Columns[i]
	}

	// Find added and modified columns
	for name, newCol := range newColumns {
		if oldCol, exists := oldColumns[name]; exists {
			if !columnsEqual(oldCol, newCol) {
				diff.ColumnChanges = append(diff.ColumnChanges, ColumnChange{
					ColumnName: name,
					Action:     ActionModify,
					OldColumn:  oldCol,
					NewColumn:  newCol,
				})
			}
		} else {
			diff.ColumnChanges = append(diff.ColumnChanges, ColumnChange{
				ColumnName: name,
				Action:     ActionAdd,
				NewColumn:  newCol,
			})
		}
	}

	// Find deleted columns
	for name, oldCol := range oldColumns {
		if _, exists := newColumns[name]; !exists {
			diff.ColumnChanges = append(diff.ColumnChanges, ColumnChange{
				ColumnName: name,
				Action:     ActionDrop,
				OldColumn:  oldCol,
			})
		}
	}

	// Compare indexes
	oldIndexes := make(map[string]*schema.Index)
	for i := range old.Indexes {
		oldIndexes[old.Indexes[i].Name] = &old.Indexes[i]
	}

	newIndexes := make(map[string]*schema.Index)
	for i := range new.Indexes {
		newIndexes[new.Indexes[i].Name] = &new.Indexes[i]
	}

	for name, newIdx := range newIndexes {
		if oldIdx, exists := oldIndexes[name]; exists {
			if !indexesEqual(oldIdx, newIdx) {
				diff.IndexChanges = append(diff.IndexChanges, IndexChange{
					IndexName: name,
					Action:    ActionModify,
					OldIndex:  oldIdx,
					NewIndex:  newIdx,
				})
			}
		} else {
			diff.IndexChanges = append(diff.IndexChanges, IndexChange{
				IndexName: name,
				Action:    ActionAdd,
				NewIndex:  newIdx,
			})
		}
	}

	for name, oldIdx := range oldIndexes {
		if _, exists := newIndexes[name]; !exists {
			diff.IndexChanges = append(diff.IndexChanges, IndexChange{
				IndexName: name,
				Action:    ActionDrop,
				OldIndex:  oldIdx,
			})
		}
	}

	// Compare foreign keys
	oldFKs := make(map[string]*schema.ForeignKey)
	for i := range old.ForeignKeys {
		oldFKs[old.ForeignKeys[i].Name] = &old.ForeignKeys[i]
	}

	newFKs := make(map[string]*schema.ForeignKey)
	for i := range new.ForeignKeys {
		newFKs[new.ForeignKeys[i].Name] = &new.ForeignKeys[i]
	}

	for name, newFK := range newFKs {
		if oldFK, exists := oldFKs[name]; exists {
			if !foreignKeysEqual(oldFK, newFK) {
				diff.ForeignKeyChanges = append(diff.ForeignKeyChanges, ForeignKeyChange{
					FKName:        name,
					Action:        ActionModify,
					OldForeignKey: oldFK,
					NewForeignKey: newFK,
				})
			}
		} else {
			diff.ForeignKeyChanges = append(diff.ForeignKeyChanges, ForeignKeyChange{
				FKName:        name,
				Action:        ActionAdd,
				NewForeignKey: newFK,
			})
		}
	}

	for name, oldFK := range oldFKs {
		if _, exists := newFKs[name]; !exists {
			diff.ForeignKeyChanges = append(diff.ForeignKeyChanges, ForeignKeyChange{
				FKName:        name,
				Action:        ActionDrop,
				OldForeignKey: oldFK,
			})
		}
	}

	// Return nil if no changes
	if len(diff.ColumnChanges) == 0 && len(diff.IndexChanges) == 0 && len(diff.ForeignKeyChanges) == 0 {
		return nil
	}

	return diff
}

func columnsEqual(a, b *schema.Column) bool {
	if a.Name != b.Name || a.Type != b.Type || a.Nullable != b.Nullable || a.AutoIncrement != b.AutoIncrement {
		return false
	}

	// Compare default values
	if (a.DefaultValue == nil) != (b.DefaultValue == nil) {
		return false
	}
	if a.DefaultValue != nil && b.DefaultValue != nil && *a.DefaultValue != *b.DefaultValue {
		return false
	}

	return true
}

func indexesEqual(a, b *schema.Index) bool {
	if a.Name != b.Name || a.Unique != b.Unique || a.Primary != b.Primary {
		return false
	}

	if len(a.Columns) != len(b.Columns) {
		return false
	}

	for i := range a.Columns {
		if a.Columns[i] != b.Columns[i] {
			return false
		}
	}

	return true
}

func foreignKeysEqual(a, b *schema.ForeignKey) bool {
	return a.Name == b.Name &&
		a.Column == b.Column &&
		a.ReferencedTable == b.ReferencedTable &&
		a.ReferencedColumn == b.ReferencedColumn &&
		a.OnDelete == b.OnDelete &&
		a.OnUpdate == b.OnUpdate
}
