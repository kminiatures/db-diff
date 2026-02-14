package generator

import (
	"fmt"
	"strings"

	"github.com/koba/db-diff/internal/diff"
	"github.com/koba/db-diff/internal/schema"
)

// DMLGenerator generates DML statements
type DMLGenerator struct {
	dbType string
}

// NewDMLGenerator creates a new DML generator
func NewDMLGenerator(dbType string) *DMLGenerator {
	return &DMLGenerator{dbType: dbType}
}

// Generate generates DML for a data diff
func (g *DMLGenerator) Generate(dataDiff *diff.DataDiff) string {
	var statements []string

	// Generate DELETE statements
	for _, row := range dataDiff.RowsDeleted {
		stmt := g.generateDelete(dataDiff.TableName, row)
		statements = append(statements, stmt)
	}

	// Generate INSERT statements
	for _, row := range dataDiff.RowsAdded {
		stmt := g.generateInsert(dataDiff.TableName, row)
		statements = append(statements, stmt)
	}

	// Generate UPDATE statements
	for _, mod := range dataDiff.RowsModified {
		stmt := g.generateUpdate(dataDiff.TableName, mod.OldRow, mod.NewRow)
		statements = append(statements, stmt)
	}

	return strings.Join(statements, "\n")
}

func (g *DMLGenerator) generateInsert(tableName string, row schema.Row) string {
	var columns []string
	var values []string

	for col, val := range row {
		columns = append(columns, g.quoteIdentifier(col))
		values = append(values, g.formatValue(val))
	}

	return fmt.Sprintf("INSERT INTO %s (%s) VALUES (%s);",
		g.quoteIdentifier(tableName),
		strings.Join(columns, ", "),
		strings.Join(values, ", "),
	)
}

func (g *DMLGenerator) generateDelete(tableName string, row schema.Row) string {
	whereClauses := g.buildWhereClause(row)
	return fmt.Sprintf("DELETE FROM %s WHERE %s;",
		g.quoteIdentifier(tableName),
		whereClauses,
	)
}

func (g *DMLGenerator) generateUpdate(tableName string, oldRow, newRow schema.Row) string {
	var setClauses []string

	for col, newVal := range newRow {
		oldVal, exists := oldRow[col]
		if !exists || !valuesEqual(oldVal, newVal) {
			setClauses = append(setClauses,
				fmt.Sprintf("%s = %s", g.quoteIdentifier(col), g.formatValue(newVal)),
			)
		}
	}

	if len(setClauses) == 0 {
		return ""
	}

	whereClauses := g.buildWhereClause(oldRow)

	return fmt.Sprintf("UPDATE %s SET %s WHERE %s;",
		g.quoteIdentifier(tableName),
		strings.Join(setClauses, ", "),
		whereClauses,
	)
}

func (g *DMLGenerator) buildWhereClause(row schema.Row) string {
	var conditions []string

	for col, val := range row {
		if val == nil {
			conditions = append(conditions,
				fmt.Sprintf("%s IS NULL", g.quoteIdentifier(col)),
			)
		} else {
			conditions = append(conditions,
				fmt.Sprintf("%s = %s", g.quoteIdentifier(col), g.formatValue(val)),
			)
		}
	}

	return strings.Join(conditions, " AND ")
}

func (g *DMLGenerator) formatValue(val interface{}) string {
	if val == nil {
		return "NULL"
	}

	switch v := val.(type) {
	case string:
		// Escape single quotes
		escaped := strings.ReplaceAll(v, "'", "''")
		return fmt.Sprintf("'%s'", escaped)
	case int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64:
		return fmt.Sprintf("%d", v)
	case float32, float64:
		return fmt.Sprintf("%f", v)
	case bool:
		if v {
			return "TRUE"
		}
		return "FALSE"
	default:
		// Fallback to string representation
		return fmt.Sprintf("'%v'", v)
	}
}

func (g *DMLGenerator) quoteIdentifier(name string) string {
	if g.dbType == "postgres" || g.dbType == "PostgreSQL" {
		return fmt.Sprintf("\"%s\"", name)
	}
	// MySQL
	return fmt.Sprintf("`%s`", name)
}

func valuesEqual(a, b interface{}) bool {
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}
	return fmt.Sprintf("%v", a) == fmt.Sprintf("%v", b)
}
