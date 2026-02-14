package schema

// Column represents a database column
type Column struct {
	Name          string  `json:"name"`
	Type          string  `json:"type"`
	Nullable      bool    `json:"nullable"`
	DefaultValue  *string `json:"default_value,omitempty"`
	AutoIncrement bool    `json:"auto_increment"`
	Position      int     `json:"position"`
}

// Index represents a database index
type Index struct {
	Name     string   `json:"name"`
	Columns  []string `json:"columns"`
	Unique   bool     `json:"unique"`
	Primary  bool     `json:"primary"`
	Type     string   `json:"type"` // e.g., BTREE, HASH
}

// ForeignKey represents a foreign key constraint
type ForeignKey struct {
	Name             string `json:"name"`
	Column           string `json:"column"`
	ReferencedTable  string `json:"referenced_table"`
	ReferencedColumn string `json:"referenced_column"`
	OnDelete         string `json:"on_delete"` // CASCADE, SET NULL, etc.
	OnUpdate         string `json:"on_update"`
}

// TableSchema represents a complete table schema
type TableSchema struct {
	Name        string       `json:"name"`
	Columns     []Column     `json:"columns"`
	Indexes     []Index      `json:"indexes"`
	ForeignKeys []ForeignKey `json:"foreign_keys"`
}

// Row represents a single row of data
type Row map[string]interface{}

// Table represents a table with its schema and data
type Table struct {
	Schema TableSchema
	Data   []Row
}
