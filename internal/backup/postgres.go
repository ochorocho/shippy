package backup

import (
	"context"
	"database/sql"
	"fmt"
	"io"
	"net"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/stdlib"

	"shippy/internal/ssh"
)

// PostgresDumper dumps a PostgreSQL database through an SSH tunnel
type PostgresDumper struct {
	db *sql.DB
}

// NewPostgresDumper creates a PostgreSQL dumper connected via SSH tunnel
func NewPostgresDumper(client *ssh.Client, creds *DatabaseCredentials) (*PostgresDumper, error) {
	port := creds.Port
	if port == 0 {
		port = 5432
	}

	addr := fmt.Sprintf("%s:%d", creds.Host, port)

	connConfig, err := pgx.ParseConfig(fmt.Sprintf(
		"postgres://%s:%s@%s:%d/%s?sslmode=disable",
		creds.User, creds.Password, creds.Host, port, creds.Name,
	))
	if err != nil {
		return nil, fmt.Errorf("failed to parse PostgreSQL config: %w", err)
	}

	connConfig.DialFunc = func(ctx context.Context, network, _ string) (net.Conn, error) {
		return client.Dial(network, addr)
	}

	db := stdlib.OpenDB(*connConfig)

	if err := db.Ping(); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to connect to PostgreSQL database '%s': %w", creds.Name, err)
	}

	return &PostgresDumper{db: db}, nil
}

// Dump writes the database dump to w, excluding tables matching the given patterns
func (d *PostgresDumper) Dump(w io.Writer, excludeTables []string, options map[string]string) error {
	fmt.Fprintf(w, "-- Shippy PostgreSQL Dump\n")
	fmt.Fprintf(w, "-- Server version: %s\n\n", d.serverVersion())
	fmt.Fprintf(w, "SET client_encoding = 'UTF8';\n")
	fmt.Fprintf(w, "SET standard_conforming_strings = on;\n\n")

	schema := "public"
	if s, ok := options["schema"]; ok {
		schema = s
	}

	tables, err := d.getTables(schema)
	if err != nil {
		return fmt.Errorf("failed to list tables: %w", err)
	}

	for _, table := range tables {
		if MatchesExcludePattern(table, excludeTables) {
			continue
		}

		if err := d.dumpTable(w, schema, table); err != nil {
			return fmt.Errorf("failed to dump table '%s': %w", table, err)
		}
	}

	return nil
}

// Close closes the database connection
func (d *PostgresDumper) Close() error {
	if d.db != nil {
		return d.db.Close()
	}
	return nil
}

func (d *PostgresDumper) serverVersion() string {
	var version string
	if err := d.db.QueryRow("SHOW server_version").Scan(&version); err != nil {
		return "unknown"
	}
	return version
}

func (d *PostgresDumper) getTables(schema string) ([]string, error) {
	rows, err := d.db.Query(
		"SELECT table_name FROM information_schema.tables WHERE table_schema = $1 AND table_type = 'BASE TABLE' ORDER BY table_name",
		schema,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tables []string
	for rows.Next() {
		var table string
		if err := rows.Scan(&table); err != nil {
			return nil, err
		}
		tables = append(tables, table)
	}
	return tables, rows.Err()
}

func (d *PostgresDumper) dumpTable(w io.Writer, schema, table string) error {
	qualifiedName := fmt.Sprintf("%s.%s", quoteIdent(schema), quoteIdent(table))

	// Get column info for CREATE TABLE
	fmt.Fprintf(w, "--\n-- Table: %s\n--\n\n", table)
	fmt.Fprintf(w, "DROP TABLE IF EXISTS %s CASCADE;\n", qualifiedName)

	createStmt, err := d.reconstructCreateTable(schema, table)
	if err != nil {
		return fmt.Errorf("create table reconstruction: %w", err)
	}
	fmt.Fprintf(w, "%s;\n\n", createStmt)

	// Dump data
	rows, err := d.db.Query(fmt.Sprintf("SELECT * FROM %s", qualifiedName))
	if err != nil {
		return fmt.Errorf("SELECT: %w", err)
	}
	defer rows.Close()

	columns, err := rows.Columns()
	if err != nil {
		return err
	}

	if len(columns) == 0 {
		return nil
	}

	values := make([]interface{}, len(columns))
	valuePtrs := make([]interface{}, len(columns))
	for i := range values {
		valuePtrs[i] = &values[i]
	}

	hasData := false
	for rows.Next() {
		if err := rows.Scan(valuePtrs...); err != nil {
			return fmt.Errorf("scan: %w", err)
		}

		if !hasData {
			fmt.Fprintf(w, "COPY %s (%s) FROM stdin;\n",
				qualifiedName,
				strings.Join(quoteIdentSlice(columns), ", "))
			hasData = true
		}

		for i, val := range values {
			if i > 0 {
				fmt.Fprintf(w, "\t")
			}
			fmt.Fprintf(w, "%s", formatPgCopyValue(val))
		}
		fmt.Fprintf(w, "\n")
	}

	if hasData {
		fmt.Fprintf(w, "\\.\n")
	}
	fmt.Fprintf(w, "\n")

	return rows.Err()
}

func (d *PostgresDumper) reconstructCreateTable(schema, table string) (string, error) {
	rows, err := d.db.Query(
		`SELECT column_name, data_type, character_maximum_length, is_nullable, column_default
		 FROM information_schema.columns
		 WHERE table_schema = $1 AND table_name = $2
		 ORDER BY ordinal_position`, schema, table)
	if err != nil {
		return "", err
	}
	defer rows.Close()

	var colDefs []string
	for rows.Next() {
		var colName, dataType, isNullable string
		var maxLen *int
		var colDefault *string

		if err := rows.Scan(&colName, &dataType, &maxLen, &isNullable, &colDefault); err != nil {
			return "", err
		}

		def := fmt.Sprintf("    %s %s", quoteIdent(colName), dataType)
		if maxLen != nil {
			def += fmt.Sprintf("(%d)", *maxLen)
		}
		if isNullable == "NO" {
			def += " NOT NULL"
		}
		if colDefault != nil {
			def += fmt.Sprintf(" DEFAULT %s", *colDefault)
		}
		colDefs = append(colDefs, def)
	}

	return fmt.Sprintf("CREATE TABLE %s.%s (\n%s\n)",
		quoteIdent(schema), quoteIdent(table),
		strings.Join(colDefs, ",\n")), rows.Err()
}

func quoteIdent(s string) string {
	return fmt.Sprintf("\"%s\"", strings.ReplaceAll(s, "\"", "\"\""))
}

func quoteIdentSlice(ss []string) []string {
	result := make([]string, len(ss))
	for i, s := range ss {
		result[i] = quoteIdent(s)
	}
	return result
}

func formatPgCopyValue(val interface{}) string {
	if val == nil {
		return "\\N"
	}
	switch v := val.(type) {
	case []byte:
		return escapePgCopyString(string(v))
	case string:
		return escapePgCopyString(v)
	default:
		return escapePgCopyString(fmt.Sprintf("%v", v))
	}
}

func escapePgCopyString(s string) string {
	r := strings.NewReplacer(
		"\\", "\\\\",
		"\t", "\\t",
		"\n", "\\n",
		"\r", "\\r",
	)
	return r.Replace(s)
}
