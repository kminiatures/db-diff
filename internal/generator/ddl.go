package generator

import (
	"fmt"
	"strings"

	"github.com/koba/db-diff/internal/diff"
	"github.com/koba/db-diff/internal/schema"
)

// DDLGenerator generates DDL statements
type DDLGenerator struct {
	dbType string
}

// NewDDLGenerator creates a new DDL generator
func NewDDLGenerator(dbType string) *DDLGenerator {
	return &DDLGenerator{dbType: dbType}
}

// Generate generates DDL for a schema diff
func (g *DDLGenerator) Generate(schemaDiff *diff.SchemaDiff) string {
	var statements []string

	switch schemaDiff.Action {
	case diff.ActionAdd:
		// Generate CREATE TABLE
		stmt := g.generateCreateTable(schemaDiff.NewSchema)
		statements = append(statements, stmt)

	case diff.ActionDrop:
		// Generate DROP TABLE
		stmt := g.generateDropTable(schemaDiff.TableName)
		statements = append(statements, stmt)

	case diff.ActionModify:
		// Generate ALTER TABLE statements

		// Drop foreign keys first
		for _, fkChange := range schemaDiff.ForeignKeyChanges {
			if fkChange.Action == diff.ActionDrop || fkChange.Action == diff.ActionModify {
				stmt := g.generateDropForeignKey(schemaDiff.TableName, fkChange.OldForeignKey.Name)
				statements = append(statements, stmt)
			}
		}

		// Drop indexes
		for _, idxChange := range schemaDiff.IndexChanges {
			if idxChange.Action == diff.ActionDrop || idxChange.Action == diff.ActionModify {
				if !idxChange.OldIndex.Primary { // Don't drop primary key index directly
					stmt := g.generateDropIndex(schemaDiff.TableName, idxChange.OldIndex.Name)
					statements = append(statements, stmt)
				}
			}
		}

		// Modify/drop/add columns
		for _, colChange := range schemaDiff.ColumnChanges {
			switch colChange.Action {
			case diff.ActionAdd:
				stmt := g.generateAddColumn(schemaDiff.TableName, colChange.NewColumn)
				statements = append(statements, stmt)
			case diff.ActionDrop:
				stmt := g.generateDropColumn(schemaDiff.TableName, colChange.ColumnName)
				statements = append(statements, stmt)
			case diff.ActionModify:
				stmt := g.generateModifyColumn(schemaDiff.TableName, colChange.NewColumn)
				statements = append(statements, stmt)
			}
		}

		// Add indexes
		for _, idxChange := range schemaDiff.IndexChanges {
			if idxChange.Action == diff.ActionAdd || idxChange.Action == diff.ActionModify {
				if !idxChange.NewIndex.Primary { // Primary key is part of CREATE TABLE
					stmt := g.generateCreateIndex(schemaDiff.TableName, idxChange.NewIndex)
					statements = append(statements, stmt)
				}
			}
		}

		// Add foreign keys
		for _, fkChange := range schemaDiff.ForeignKeyChanges {
			if fkChange.Action == diff.ActionAdd || fkChange.Action == diff.ActionModify {
				stmt := g.generateAddForeignKey(schemaDiff.TableName, fkChange.NewForeignKey)
				statements = append(statements, stmt)
			}
		}
	}

	return strings.Join(statements, "\n")
}

func (g *DDLGenerator) generateCreateTable(tableSchema *schema.TableSchema) string {
	var parts []string

	// Column definitions
	for _, col := range tableSchema.Columns {
		parts = append(parts, g.columnDefinition(&col))
	}

	// Primary key
	for _, idx := range tableSchema.Indexes {
		if idx.Primary {
			pkCols := strings.Join(g.quoteIdentifiers(idx.Columns), ", ")
			parts = append(parts, fmt.Sprintf("PRIMARY KEY (%s)", pkCols))
			break
		}
	}

	// Foreign keys
	for _, fk := range tableSchema.ForeignKeys {
		fkDef := fmt.Sprintf("CONSTRAINT %s FOREIGN KEY (%s) REFERENCES %s(%s)",
			g.quoteIdentifier(fk.Name),
			g.quoteIdentifier(fk.Column),
			g.quoteIdentifier(fk.ReferencedTable),
			g.quoteIdentifier(fk.ReferencedColumn),
		)
		if fk.OnDelete != "" {
			fkDef += fmt.Sprintf(" ON DELETE %s", fk.OnDelete)
		}
		if fk.OnUpdate != "" {
			fkDef += fmt.Sprintf(" ON UPDATE %s", fk.OnUpdate)
		}
		parts = append(parts, fkDef)
	}

	tableName := g.quoteIdentifier(tableSchema.Name)
	return fmt.Sprintf("CREATE TABLE %s (\n  %s\n);", tableName, strings.Join(parts, ",\n  "))
}

func (g *DDLGenerator) generateDropTable(tableName string) string {
	return fmt.Sprintf("DROP TABLE %s;", g.quoteIdentifier(tableName))
}

func (g *DDLGenerator) generateAddColumn(tableName string, col *schema.Column) string {
	return fmt.Sprintf("ALTER TABLE %s ADD COLUMN %s;",
		g.quoteIdentifier(tableName),
		g.columnDefinition(col),
	)
}

func (g *DDLGenerator) generateDropColumn(tableName, columnName string) string {
	return fmt.Sprintf("ALTER TABLE %s DROP COLUMN %s;",
		g.quoteIdentifier(tableName),
		g.quoteIdentifier(columnName),
	)
}

func (g *DDLGenerator) generateModifyColumn(tableName string, col *schema.Column) string {
	if g.dbType == "postgres" || g.dbType == "PostgreSQL" {
		return fmt.Sprintf("ALTER TABLE %s ALTER COLUMN %s TYPE %s;",
			g.quoteIdentifier(tableName),
			g.quoteIdentifier(col.Name),
			col.Type,
		)
	}
	// MySQL
	return fmt.Sprintf("ALTER TABLE %s MODIFY COLUMN %s;",
		g.quoteIdentifier(tableName),
		g.columnDefinition(col),
	)
}

func (g *DDLGenerator) generateCreateIndex(tableName string, idx *schema.Index) string {
	indexType := ""
	if idx.Unique {
		indexType = "UNIQUE "
	}

	columns := strings.Join(g.quoteIdentifiers(idx.Columns), ", ")
	return fmt.Sprintf("CREATE %sINDEX %s ON %s (%s);",
		indexType,
		g.quoteIdentifier(idx.Name),
		g.quoteIdentifier(tableName),
		columns,
	)
}

func (g *DDLGenerator) generateDropIndex(tableName, indexName string) string {
	if g.dbType == "postgres" || g.dbType == "PostgreSQL" {
		return fmt.Sprintf("DROP INDEX %s;", g.quoteIdentifier(indexName))
	}
	// MySQL
	return fmt.Sprintf("DROP INDEX %s ON %s;",
		g.quoteIdentifier(indexName),
		g.quoteIdentifier(tableName),
	)
}

func (g *DDLGenerator) generateAddForeignKey(tableName string, fk *schema.ForeignKey) string {
	fkDef := fmt.Sprintf("ALTER TABLE %s ADD CONSTRAINT %s FOREIGN KEY (%s) REFERENCES %s(%s)",
		g.quoteIdentifier(tableName),
		g.quoteIdentifier(fk.Name),
		g.quoteIdentifier(fk.Column),
		g.quoteIdentifier(fk.ReferencedTable),
		g.quoteIdentifier(fk.ReferencedColumn),
	)
	if fk.OnDelete != "" {
		fkDef += fmt.Sprintf(" ON DELETE %s", fk.OnDelete)
	}
	if fk.OnUpdate != "" {
		fkDef += fmt.Sprintf(" ON UPDATE %s", fk.OnUpdate)
	}
	return fkDef + ";"
}

func (g *DDLGenerator) generateDropForeignKey(tableName, fkName string) string {
	if g.dbType == "postgres" || g.dbType == "PostgreSQL" {
		return fmt.Sprintf("ALTER TABLE %s DROP CONSTRAINT %s;",
			g.quoteIdentifier(tableName),
			g.quoteIdentifier(fkName),
		)
	}
	// MySQL
	return fmt.Sprintf("ALTER TABLE %s DROP FOREIGN KEY %s;",
		g.quoteIdentifier(tableName),
		g.quoteIdentifier(fkName),
	)
}

func (g *DDLGenerator) columnDefinition(col *schema.Column) string {
	def := g.quoteIdentifier(col.Name) + " " + col.Type

	if !col.Nullable {
		def += " NOT NULL"
	}

	if col.DefaultValue != nil {
		def += fmt.Sprintf(" DEFAULT %s", *col.DefaultValue)
	}

	if col.AutoIncrement {
		if g.dbType == "postgres" || g.dbType == "PostgreSQL" {
			// PostgreSQL uses SERIAL or IDENTITY
		} else {
			// MySQL
			def += " AUTO_INCREMENT"
		}
	}

	return def
}

func (g *DDLGenerator) quoteIdentifier(name string) string {
	if g.dbType == "postgres" || g.dbType == "PostgreSQL" {
		return fmt.Sprintf("\"%s\"", name)
	}
	// MySQL
	return fmt.Sprintf("`%s`", name)
}

func (g *DDLGenerator) quoteIdentifiers(names []string) []string {
	quoted := make([]string, len(names))
	for i, name := range names {
		quoted[i] = g.quoteIdentifier(name)
	}
	return quoted
}
