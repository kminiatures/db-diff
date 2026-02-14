package generator

import (
	"strings"

	"github.com/koba/db-diff/internal/diff"
)

// GenerateSQL generates migration SQL from a diff result
func GenerateSQL(result *diff.DiffResult, dbType string) string {
	var sqlStatements []string

	// Generate DDL statements
	ddlGen := NewDDLGenerator(dbType)
	for _, schemaDiff := range result.SchemaDiffs {
		sql := ddlGen.Generate(schemaDiff)
		if sql != "" {
			sqlStatements = append(sqlStatements, sql)
		}
	}

	// Generate DML statements
	dmlGen := NewDMLGenerator(dbType)
	for _, dataDiff := range result.DataDiffs {
		sql := dmlGen.Generate(dataDiff)
		if sql != "" {
			sqlStatements = append(sqlStatements, sql)
		}
	}

	return strings.Join(sqlStatements, "\n\n")
}
