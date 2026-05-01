package backup

import (
	"context"
	"database/sql"
	"fmt"
	"io"
	"net"
	"strings"
	"sync/atomic"

	mysqldriver "github.com/go-sql-driver/mysql"

	"shippy/internal/ssh"
)

// dialerCounter ensures unique dialer names for concurrent use
var dialerCounter atomic.Int64

// MySQLDumper dumps a MySQL/MariaDB database through an SSH tunnel
type MySQLDumper struct {
	db         *sql.DB
	dialerName string
}

// NewMySQLDumper creates a MySQL dumper connected via SSH tunnel
func NewMySQLDumper(client *ssh.Client, creds *DatabaseCredentials) (*MySQLDumper, error) {
	port := creds.Port
	if port == 0 {
		port = 3306
	}

	addr := fmt.Sprintf("%s:%d", creds.Host, port)
	dialerName := fmt.Sprintf("shippy-ssh-%d", dialerCounter.Add(1))

	mysqldriver.RegisterDialContext(dialerName, func(ctx context.Context, _ string) (net.Conn, error) {
		return client.Dial("tcp", addr)
	})

	dsn := fmt.Sprintf("%s:%s@%s(%s)/%s?charset=utf8mb4&parseTime=true",
		creds.User, creds.Password, dialerName, addr, creds.Name)

	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open MySQL connection: %w", err)
	}

	if err := db.Ping(); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("failed to connect to MySQL database '%s': %w", creds.Name, err)
	}

	return &MySQLDumper{db: db, dialerName: dialerName}, nil
}

// Dump writes the database dump to w, excluding tables matching the given patterns
func (d *MySQLDumper) Dump(w io.Writer, excludeTables []string, options map[string]string) error {
	// Write header
	fmt.Fprintf(w, "-- Shippy MySQL Dump\n")
	fmt.Fprintf(w, "-- Server version: %s\n\n", d.serverVersion())
	fmt.Fprintf(w, "SET NAMES utf8mb4;\n")
	fmt.Fprintf(w, "SET FOREIGN_KEY_CHECKS = 0;\n\n")

	// Check for single_transaction option
	if options["single_transaction"] == "true" {
		if _, err := d.db.Exec("SET SESSION TRANSACTION ISOLATION LEVEL REPEATABLE READ"); err != nil {
			return fmt.Errorf("failed to set transaction isolation: %w", err)
		}
		tx, err := d.db.Begin()
		if err != nil {
			return fmt.Errorf("failed to start transaction: %w", err)
		}
		defer tx.Rollback()
	}

	// Get table list
	tables, err := d.getTables()
	if err != nil {
		return fmt.Errorf("failed to list tables: %w", err)
	}

	for _, table := range tables {
		if MatchesExcludePattern(table, excludeTables) {
			continue
		}

		if err := d.dumpTable(w, table); err != nil {
			return fmt.Errorf("failed to dump table '%s': %w", table, err)
		}
	}

	fmt.Fprintf(w, "SET FOREIGN_KEY_CHECKS = 1;\n")
	return nil
}

// Close closes the database connection
func (d *MySQLDumper) Close() error {
	if d.db != nil {
		return d.db.Close()
	}
	return nil
}

func (d *MySQLDumper) serverVersion() string {
	var version string
	if err := d.db.QueryRow("SELECT VERSION()").Scan(&version); err != nil {
		return "unknown"
	}
	return version
}

func (d *MySQLDumper) getTables() ([]string, error) {
	rows, err := d.db.Query("SHOW TABLES")
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

func (d *MySQLDumper) dumpTable(w io.Writer, table string) error {
	// Write CREATE TABLE
	var tableName, createStmt string
	err := d.db.QueryRow(fmt.Sprintf("SHOW CREATE TABLE `%s`", table)).Scan(&tableName, &createStmt)
	if err != nil {
		return fmt.Errorf("SHOW CREATE TABLE: %w", err)
	}

	fmt.Fprintf(w, "--\n-- Table structure for `%s`\n--\n\n", table)
	fmt.Fprintf(w, "DROP TABLE IF EXISTS `%s`;\n", table)
	fmt.Fprintf(w, "%s;\n\n", createStmt)

	// Dump data
	rows, err := d.db.Query(fmt.Sprintf("SELECT * FROM `%s`", table))
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

	fmt.Fprintf(w, "--\n-- Data for `%s`\n--\n\n", table)

	values := make([]interface{}, len(columns))
	valuePtrs := make([]interface{}, len(columns))
	for i := range values {
		valuePtrs[i] = &values[i]
	}

	batchSize := 100
	insertCount := 0

	for rows.Next() {
		if err := rows.Scan(valuePtrs...); err != nil {
			return fmt.Errorf("scan: %w", err)
		}

		if insertCount%batchSize == 0 {
			if insertCount > 0 {
				fmt.Fprintf(w, ";\n")
			}
			fmt.Fprintf(w, "INSERT INTO `%s` (`%s`) VALUES\n", table, strings.Join(columns, "`, `"))
		} else {
			fmt.Fprintf(w, ",\n")
		}

		fmt.Fprintf(w, "(")
		for i, val := range values {
			if i > 0 {
				fmt.Fprintf(w, ", ")
			}
			fmt.Fprintf(w, "%s", formatMySQLValue(val))
		}
		fmt.Fprintf(w, ")")

		insertCount++
	}

	if insertCount > 0 {
		fmt.Fprintf(w, ";\n")
	}
	fmt.Fprintf(w, "\n")

	return rows.Err()
}

func formatMySQLValue(val interface{}) string {
	if val == nil {
		return "NULL"
	}
	switch v := val.(type) {
	case []byte:
		return fmt.Sprintf("'%s'", escapeMySQLString(string(v)))
	case string:
		return fmt.Sprintf("'%s'", escapeMySQLString(v))
	case int64:
		return fmt.Sprintf("%d", v)
	case float64:
		return fmt.Sprintf("%g", v)
	case bool:
		if v {
			return "1"
		}
		return "0"
	default:
		return fmt.Sprintf("'%s'", escapeMySQLString(fmt.Sprintf("%v", v)))
	}
}

func escapeMySQLString(s string) string {
	r := strings.NewReplacer(
		"\\", "\\\\",
		"'", "\\'",
		"\x00", "\\0",
		"\n", "\\n",
		"\r", "\\r",
		"\x1a", "\\Z",
	)
	return r.Replace(s)
}
