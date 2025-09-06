package sqb

import (
	"context"
	"fmt"
	"strings"

	"github.com/jmoiron/sqlx"
)

func ValidateColumnsExist(ctx context.Context, db *sqlx.DB, d Dialect, table string, cols []string) error {
	colMap, err := listColumns(ctx, db, d, table)
	if err != nil { return err }
	var missing []string
	for _, c := range cols {
		if _, ok := colMap[strings.ToLower(c)]; !ok {
			missing = append(missing, c)
		}
	}
	if len(missing) > 0 {
		return fmt.Errorf("sqb: missing columns in %s: %s", table, strings.Join(missing, ", "))
	}
	return nil
}

func listColumns(ctx context.Context, db *sqlx.DB, d Dialect, fqn string) (map[string]string, error) {
	schema, table := splitFQNForIntrospection(d, fqn)

	switch d.Name() {
	case "postgres":
		q := `
		SELECT
		  c.column_name AS name,
		  CASE
		    WHEN c.data_type = 'character varying'
		      THEN 'VARCHAR(' || c.character_maximum_length || ')'
		    WHEN c.data_type = 'USER-DEFINED' THEN UPPER(c.udt_name)
		    ELSE UPPER(c.data_type)
		  END AS type
		FROM information_schema.columns c
		WHERE c.table_schema = $1 AND c.table_name = $2
		ORDER BY c.ordinal_position`
		type row struct{ Name, Type string }
		var rows []row
		if err := db.SelectContext(ctx, &rows, q, schema, table); err != nil { return nil, err }
		out := make(map[string]string, len(rows))
		for _, r := range rows { out[strings.ToLower(r.Name)] = strings.ToUpper(strings.TrimSpace(r.Type)) }
		return out, nil

	case "mysql":
		q := `
		SELECT
		  COLUMN_NAME AS name,
		  UPPER(CASE
		    WHEN DATA_TYPE IN ('varchar','varbinary') AND CHARACTER_MAXIMUM_LENGTH IS NOT NULL
		      THEN CONCAT(UCASE(DATA_TYPE),'(',CHARACTER_MAXIMUM_LENGTH,')')
		    ELSE DATA_TYPE
		  END) AS type
		FROM information_schema.columns
		WHERE (TABLE_SCHEMA = IF(?='', DATABASE(), ?)) AND TABLE_NAME = ?
		ORDER BY ORDINAL_POSITION`
		type row struct{ Name, Type string }
		var rows []row
		if err := db.SelectContext(ctx, &rows, q, schema, schema, table); err != nil { return nil, err }
		out := make(map[string]string, len(rows))
		for _, r := range rows { out[strings.ToLower(r.Name)] = strings.ToUpper(strings.TrimSpace(r.Type)) }
		return out, nil

	case "sqlite":
		q := fmt.Sprintf("PRAGMA %s.table_info(%s)", schema, table)
		type srow struct{ Name, Type string }
		var rows []srow
		if err := db.SelectContext(ctx, &rows, q); err != nil { return nil, err }
		out := make(map[string]string, len(rows))
		for _, r := range rows { out[strings.ToLower(r.Name)] = strings.ToUpper(strings.TrimSpace(r.Type)) }
		return out, nil

	case "sqlserver":
		q := `
		SELECT
		  c.COLUMN_NAME AS name,
		  UPPER(CASE
		    WHEN c.DATA_TYPE IN ('varchar','nvarchar','varbinary') AND c.CHARACTER_MAXIMUM_LENGTH IS NOT NULL
		      THEN c.DATA_TYPE + '(' + IIF(c.CHARACTER_MAXIMUM_LENGTH=-1,'MAX',CONVERT(varchar(10),c.CHARACTER_MAXIMUM_LENGTH)) + ')'
		    ELSE c.DATA_TYPE
		  END) AS type
		FROM INFORMATION_SCHEMA.COLUMNS c
		WHERE c.TABLE_SCHEMA = @p1 AND c.TABLE_NAME = @p2
		ORDER BY c.ORDINAL_POSITION`
		type row struct{ Name, Type string }
		var rows []row
		if err := db.SelectContext(ctx, &rows, q, schema, table); err != nil { return nil, err }
		out := make(map[string]string, len(rows))
		for _, r := range rows { out[strings.ToLower(r.Name)] = strings.ToUpper(strings.TrimSpace(r.Type)) }
		return out, nil
	}
	return map[string]string{}, nil
}

func splitFQNForIntrospection(d Dialect, fqn string) (schema, table string) {
	parts := strings.SplitN(strings.ReplaceAll(fqn, `"`, ""), ".", 2)
	if len(parts) == 2 { return parts[0], parts[1] }
	switch d.Name() {
	case "postgres": return "public", fqn
	case "mysql":    return "", fqn
	case "sqlite":   return "main", fqn
	case "sqlserver":return "dbo", fqn
	default:         return "", fqn
	}
}
