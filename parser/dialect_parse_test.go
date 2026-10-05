package parser

import (
	"testing"

	"github.com/sqls-server/sqls/ast"
)

func TestDialectSyntax_Parse(t *testing.T) {
	tests := []struct {
		name          string
		query         string
		expectedStmts int
		checkNodes    func(t *testing.T, stmt ast.TokenList)
	}{
		// --- ClickHouse ---
		{
			name:          "ClickHouse: FINAL modifier",
			query:         "SELECT * FROM users FINAL WHERE id = 1",
			expectedStmts: 1,
			checkNodes: func(t *testing.T, stmt ast.TokenList) {
				s := stmt.String()
				if s != "SELECT * FROM users FINAL WHERE id = 1" {
					t.Errorf("unexpected string: %s", s)
				}
			},
		},
		{
			name:          "ClickHouse: PREWHERE and WHERE clauses",
			query:         "SELECT * FROM hits PREWHERE UserID = 123 WHERE CounterID = 456",
			expectedStmts: 1,
			checkNodes: func(t *testing.T, stmt ast.TokenList) {
				s := stmt.String()
				if s != "SELECT * FROM hits PREWHERE UserID = 123 WHERE CounterID = 456" {
					t.Errorf("unexpected string: %s", s)
				}
			},
		},
		{
			name:          "ClickHouse: SETTINGS and FORMAT",
			query:         "SELECT id, name FROM users SETTINGS max_threads = 2 FORMAT JSON",
			expectedStmts: 1,
			checkNodes: func(t *testing.T, stmt ast.TokenList) {
				s := stmt.String()
				if s != "SELECT id, name FROM users SETTINGS max_threads = 2 FORMAT JSON" {
					t.Errorf("unexpected string: %s", s)
				}
			},
		},

		// --- PostgreSQL ---
		{
			name:          "PostgreSQL: ON CONFLICT DO UPDATE",
			query:         "INSERT INTO users (id, name) VALUES (1, 'alice') ON CONFLICT (id) DO UPDATE SET name = 'bob'",
			expectedStmts: 1,
			checkNodes: func(t *testing.T, stmt ast.TokenList) {
				s := stmt.String()
				if s != "INSERT INTO users (id, name) VALUES (1, 'alice') ON CONFLICT (id) DO UPDATE SET name = 'bob'" {
					t.Errorf("unexpected string: %s", s)
				}
			},
		},
		{
			name:          "PostgreSQL: RETURNING clause",
			query:         "INSERT INTO users (name) VALUES ('alice') RETURNING id, created_at",
			expectedStmts: 1,
			checkNodes: func(t *testing.T, stmt ast.TokenList) {
				s := stmt.String()
				if s != "INSERT INTO users (name) VALUES ('alice') RETURNING id, created_at" {
					t.Errorf("unexpected string: %s", s)
				}
			},
		},
		{
			name:          "PostgreSQL: DISTINCT ON",
			query:         "SELECT DISTINCT ON (dept) dept, salary FROM employees ORDER BY dept, salary DESC",
			expectedStmts: 1,
			checkNodes: func(t *testing.T, stmt ast.TokenList) {
				s := stmt.String()
				if s != "SELECT DISTINCT ON (dept) dept, salary FROM employees ORDER BY dept, salary DESC" {
					t.Errorf("unexpected string: %s", s)
				}
			},
		},
		{
			name:          "PostgreSQL: ILIKE and FOR UPDATE SKIP LOCKED",
			query:         "SELECT * FROM orders WHERE status ILIKE 'pending%' FOR UPDATE SKIP LOCKED",
			expectedStmts: 1,
			checkNodes: func(t *testing.T, stmt ast.TokenList) {
				s := stmt.String()
				if s != "SELECT * FROM orders WHERE status ILIKE 'pending%' FOR UPDATE SKIP LOCKED" {
					t.Errorf("unexpected string: %s", s)
				}
			},
		},

		// --- MySQL ---
		{
			name:          "MySQL: ON DUPLICATE KEY UPDATE",
			query:         "INSERT INTO stats (id, cnt) VALUES (1, 1) ON DUPLICATE KEY UPDATE cnt = cnt + 1",
			expectedStmts: 1,
			checkNodes: func(t *testing.T, stmt ast.TokenList) {
				s := stmt.String()
				if s != "INSERT INTO stats (id, cnt) VALUES (1, 1) ON DUPLICATE KEY UPDATE cnt = cnt + 1" {
					t.Errorf("unexpected string: %s", s)
				}
			},
		},
		{
			name:          "MySQL: FORCE INDEX hint",
			query:         "SELECT * FROM users FORCE INDEX (idx_email) WHERE email = 'test@example.com'",
			expectedStmts: 1,
			checkNodes: func(t *testing.T, stmt ast.TokenList) {
				s := stmt.String()
				if s != "SELECT * FROM users FORCE INDEX (idx_email) WHERE email = 'test@example.com'" {
					t.Errorf("unexpected string: %s", s)
				}
			},
		},
		{
			name:          "MySQL: REPLACE INTO statement",
			query:         "REPLACE INTO users (id, name) VALUES (1, 'alice')",
			expectedStmts: 1,
			checkNodes: func(t *testing.T, stmt ast.TokenList) {
				s := stmt.String()
				if s != "REPLACE INTO users (id, name) VALUES (1, 'alice')" {
					t.Errorf("unexpected string: %s", s)
				}
			},
		},

		// --- SQLite ---
		{
			name:          "SQLite: PRAGMA table_info",
			query:         "PRAGMA table_info(users)",
			expectedStmts: 1,
			checkNodes: func(t *testing.T, stmt ast.TokenList) {
				s := stmt.String()
				if s != "PRAGMA table_info(users)" {
					t.Errorf("unexpected string: %s", s)
				}
			},
		},
		{
			name:          "SQLite: ON CONFLICT DO NOTHING RETURNING",
			query:         "INSERT INTO users (id) VALUES (1) ON CONFLICT DO NOTHING RETURNING id",
			expectedStmts: 1,
			checkNodes: func(t *testing.T, stmt ast.TokenList) {
				s := stmt.String()
				if s != "INSERT INTO users (id) VALUES (1) ON CONFLICT DO NOTHING RETURNING id" {
					t.Errorf("unexpected string: %s", s)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parsed, err := Parse(tt.query)
			if err != nil {
				t.Fatalf("Parse error: %v", err)
			}
			stmts := parsed.GetTokens()
			if len(stmts) != tt.expectedStmts {
				t.Fatalf("expected %d statements, got %d", tt.expectedStmts, len(stmts))
			}
			stmt, ok := stmts[0].(ast.TokenList)
			if !ok {
				t.Fatalf("statement is not ast.TokenList: %T", stmts[0])
			}
			if tt.checkNodes != nil {
				tt.checkNodes(t, stmt)
			}
		})
	}
}

func TestDialectSyntax_ParseSequentialNoSemicolon(t *testing.T) {
	tests := []struct {
		name          string
		query         string
		expectedStmts int
	}{
		{
			name: "ClickHouse: sequential with FINAL without semicolon",
			query: "SELECT * FROM users FINAL\nSELECT * FROM hits",
			expectedStmts: 2,
		},
		{
			name: "ClickHouse: sequential with FORMAT without semicolon",
			query: "SELECT * FROM users FORMAT JSON\nSELECT * FROM hits",
			expectedStmts: 2,
		},
		{
			name: "PostgreSQL: INSERT with RETURNING followed by SELECT without semicolon",
			query: "INSERT INTO users (name) VALUES ('alice') RETURNING id\nSELECT * FROM users",
			expectedStmts: 2,
		},
		{
			name: "MySQL: INSERT ON DUPLICATE KEY UPDATE followed by SELECT without semicolon",
			query: "INSERT INTO stats (id) VALUES (1) ON DUPLICATE KEY UPDATE cnt = cnt + 1\nSELECT * FROM stats",
			expectedStmts: 2,
		},
		{
			name: "SQLite: PRAGMA followed by SELECT without semicolon",
			query: "PRAGMA table_info(users)\nSELECT * FROM users",
			expectedStmts: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parsed, err := Parse(tt.query)
			if err != nil {
				t.Fatalf("Parse error: %v", err)
			}
			stmts := parsed.GetTokens()
			if len(stmts) != tt.expectedStmts {
				t.Fatalf("expected %d statements, got %d. Tokens: %v", tt.expectedStmts, len(stmts), stmts)
			}
		})
	}
}
