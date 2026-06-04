package backup

import (
	"testing"
)

func TestMatchesExcludePattern(t *testing.T) {
	tests := []struct {
		name     string
		table    string
		patterns []string
		want     bool
	}{
		{
			name:     "exact match",
			table:    "sys_log",
			patterns: []string{"sys_log"},
			want:     true,
		},
		{
			name:     "wildcard suffix",
			table:    "cache_pages",
			patterns: []string{"cache_*"},
			want:     true,
		},
		{
			name:     "wildcard prefix",
			table:    "tx_myext_domain_model_item",
			patterns: []string{"tx_myext_*"},
			want:     true,
		},
		{
			name:     "no match",
			table:    "pages",
			patterns: []string{"cache_*", "sys_log"},
			want:     false,
		},
		{
			name:     "multiple patterns first matches",
			table:    "cf_cache_hash",
			patterns: []string{"cf_*", "cache_*"},
			want:     true,
		},
		{
			name:     "multiple patterns second matches",
			table:    "be_sessions",
			patterns: []string{"cache_*", "be_sessions"},
			want:     true,
		},
		{
			name:     "empty patterns",
			table:    "pages",
			patterns: []string{},
			want:     false,
		},
		{
			name:     "nil patterns",
			table:    "pages",
			patterns: nil,
			want:     false,
		},
		{
			name:     "question mark wildcard",
			table:    "cache_a",
			patterns: []string{"cache_?"},
			want:     true,
		},
		{
			name:     "question mark no match longer",
			table:    "cache_ab",
			patterns: []string{"cache_?"},
			want:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := MatchesExcludePattern(tt.table, tt.patterns)
			if got != tt.want {
				t.Errorf("MatchesExcludePattern(%q, %v) = %v, want %v", tt.table, tt.patterns, got, tt.want)
			}
		})
	}
}

func TestNormalizeDriver(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"mysql", "mysql"},
		{"MySQL", "mysql"},
		{"mysqli", "mysql"},
		{"pdo_mysql", "mysql"},
		{"mariadb", "mysql"},
		{"MariaDB", "mysql"},
		{"postgresql", "postgresql"},
		{"postgres", "postgresql"},
		{"pgsql", "postgresql"},
		{"pdo_pgsql", "postgresql"},
		{"sqlite", "sqlite"},
		{"sqlite3", "sqlite"},
		{"pdo_sqlite", "sqlite"},
		{"unknown", "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := normalizeDriver(tt.input)
			if got != tt.want {
				t.Errorf("normalizeDriver(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestParseConsoleCredentials(t *testing.T) {
	tests := []struct {
		name    string
		output  string
		wantErr bool
		want    DatabaseCredentials
	}{
		{
			name:   "typical v14 output",
			output: `{"charset":"utf8mb4","dbname":"app","driver":"mysqli","host":"db","password":"secret","port":3306,"user":"app_user"}`,
			want: DatabaseCredentials{
				Driver:   "mysql",
				Host:     "db",
				Port:     3306,
				Name:     "app",
				User:     "app_user",
				Password: "secret",
			},
		},
		{
			name:   "json wrapped in surrounding noise",
			output: "PHP Deprecated: something\n{\"dbname\":\"app\",\"driver\":\"pdo_pgsql\",\"host\":\"127.0.0.1\",\"port\":5432,\"user\":\"u\",\"password\":\"p\"}\n",
			want: DatabaseCredentials{
				Driver:   "postgresql",
				Host:     "127.0.0.1",
				Port:     5432,
				Name:     "app",
				User:     "u",
				Password: "p",
			},
		},
		{
			name:    "missing dbname",
			output:  `{"driver":"mysqli","host":"db","port":3306}`,
			wantErr: true,
		},
		{
			name:    "empty output",
			output:  "",
			wantErr: true,
		},
		{
			name:    "no json object",
			output:  "Command \"configuration:show\" is not defined.",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseConsoleCredentials(tt.output)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("parseConsoleCredentials(%q) expected error, got nil", tt.output)
				}
				return
			}
			if err != nil {
				t.Fatalf("parseConsoleCredentials(%q) unexpected error: %v", tt.output, err)
			}
			if *got != tt.want {
				t.Errorf("parseConsoleCredentials(%q) = %+v, want %+v", tt.output, *got, tt.want)
			}
		})
	}
}

func TestFormatMySQLValue(t *testing.T) {
	tests := []struct {
		name  string
		input interface{}
		want  string
	}{
		{"nil", nil, "NULL"},
		{"string", "hello", "'hello'"},
		{"string with quote", "it's", "'it\\'s'"},
		{"string with backslash", "path\\to", "'path\\\\to'"},
		{"string with newline", "line1\nline2", "'line1\\nline2'"},
		{"int64", int64(42), "42"},
		{"float64", float64(3.14), "3.14"},
		{"bool true", true, "1"},
		{"bool false", false, "0"},
		{"bytes", []byte("data"), "'data'"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := formatMySQLValue(tt.input)
			if got != tt.want {
				t.Errorf("formatMySQLValue(%v) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestFormatPgCopyValue(t *testing.T) {
	tests := []struct {
		name  string
		input interface{}
		want  string
	}{
		{"nil", nil, "\\N"},
		{"string", "hello", "hello"},
		{"string with tab", "col1\tcol2", "col1\\tcol2"},
		{"string with newline", "line1\nline2", "line1\\nline2"},
		{"string with backslash", "path\\to", "path\\\\to"},
		{"bytes", []byte("data"), "data"},
		{"int64", int64(42), "42"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := formatPgCopyValue(tt.input)
			if got != tt.want {
				t.Errorf("formatPgCopyValue(%v) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestEscapeMySQLString(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"hello", "hello"},
		{"it's", "it\\'s"},
		{"back\\slash", "back\\\\slash"},
		{"null\x00byte", "null\\0byte"},
		{"new\nline", "new\\nline"},
		{"cr\rreturn", "cr\\rreturn"},
		{"sub\x1astitute", "sub\\Zstitute"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := escapeMySQLString(tt.input)
			if got != tt.want {
				t.Errorf("escapeMySQLString(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}
