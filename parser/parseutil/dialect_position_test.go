package parseutil

import (
	"testing"

	"github.com/sqls-server/sqls/parser"
	"github.com/sqls-server/sqls/token"
)

func TestDialectSyntaxPosition(t *testing.T) {
	tests := []struct {
		name string
		text string
		pos  token.Pos
		want SyntaxPosition
	}{
		// --- ClickHouse ---
		{
			name: "ClickHouse: after PREWHERE expects WhereCondition",
			text: "SELECT * FROM hits PREWHERE ",
			pos: token.Pos{
				Line: 0,
				Col:  28,
			},
			want: WhereCondition,
		},

		// --- PostgreSQL ---
		{
			name: "PostgreSQL: after RETURNING expects SelectExpr",
			text: "INSERT INTO users (name) VALUES ('alice') RETURNING ",
			pos: token.Pos{
				Line: 0,
				Col:  52,
			},
			want: SelectExpr,
		},
		{
			name: "PostgreSQL: inside DISTINCT ON parens expects ColName",
			text: "SELECT DISTINCT ON (",
			pos: token.Pos{
				Line: 0,
				Col:  20,
			},
			want: ColName,
		},
		{
			name: "PostgreSQL: inside ON CONFLICT parens expects ColName",
			text: "INSERT INTO users (id) VALUES (1) ON CONFLICT (",
			pos: token.Pos{
				Line: 0,
				Col:  47,
			},
			want: ColName,
		},

		// --- MySQL ---
		{
			name: "MySQL: inside FORCE INDEX parens expects ColName",
			text: "SELECT * FROM users FORCE INDEX (",
			pos: token.Pos{
				Line: 0,
				Col:  33,
			},
			want: ColName,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parsed, err := parser.Parse(tt.text)
			if err != nil {
				t.Fatalf("parse error: %v", err)
			}
			nw := NewNodeWalker(parsed, tt.pos)
			got := CheckSyntaxPosition(nw)
			if got != tt.want {
				t.Errorf("syntax position mismatch: got %v, want %v", got, tt.want)
			}
		})
	}
}
