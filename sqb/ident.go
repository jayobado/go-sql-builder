package sqb

import "strings"

func SplitSchemaTable(d Dialect, qualified string) (schema, table string) {
	qualified = strings.ReplaceAll(qualified, `"`, "")
	if strings.Contains(qualified, ".") {
		parts := strings.SplitN(qualified, ".", 2)
		return parts[0], parts[1]
	}
	switch d.Name() {
	case "postgres": return "public", qualified
	case "mysql":    return "", qualified // use current DB
	case "sqlite":   return "main", qualified
	case "sqlserver":return "dbo", qualified
	default:         return "", qualified
	}
}

// Quote schema.table using the builder’s quoting rules
func QuoteFQN(d Dialect, schema, table string) string {
	if schema == "" || d.Name() == "mysql" { // MySQL rarely uses schema separate from DB
		return d.QuoteIdent(table)
	}
	return d.QuoteIdent(schema) + "." + d.QuoteIdent(table)
}