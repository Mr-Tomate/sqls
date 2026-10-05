package completer

import (
	"reflect"
	"strings"
	"testing"

	"github.com/sqls-server/sqls/internal/lsp"
)

func TestGetBeforeCursorText(t *testing.T) {
	input := `SELECT
a, b, c
FROM
hogetable
`
	tests := []struct {
		in   string
		line int
		char int
		out  string
	}{
		{input, 1, 2, "SE"},
		{input, 2, 3, "SELECT\na, "},
		{input, 3, 4, "SELECT\na, b, c\nFROM"},
		{input, 4, 5, "SELECT\na, b, c\nFROM\nhoget"},
		{"select 'テスト', ci", 1, 16, "select 'テスト', ci"},
		{"select '😀', ci", 1, 15, "select '😀', ci"},
		{"select '😀', ci", 1, 13, "select '😀', "},
		{"select 1", 1, 100, "select 1"},
	}
	for _, tt := range tests {
		got := getBeforeCursorText(tt.in, tt.line, tt.char)
		if tt.out != got {
			t.Errorf("want %#v, got %#v", tt.out, got)
		}
	}
}

func TestGetLastWord(t *testing.T) {
	input := `SELECT
    a, b, c
FROM  
    hogetable
`
	tests := []struct {
		name string
		in   string
		line int
		char int
		out  string
	}{
		{"", "SELECT  FROM def", 1, 7, ""},
		{"", input, 1, 2, "SE"},
		{"", input, 2, 3, ""},
		{"", input, 3, 4, "FROM"},
		{"", input, 3, 6, ""},
		{"", input, 4, 5, "h"},
		{"", "`ident", 1, 6, "`ident"},
		{"", "parent.`ident", 1, 13, "`ident"},
		{"", "`parent`.`ident", 1, 15, "`ident"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := getLastWord(tt.in, tt.line, tt.char)
			if tt.out != got {
				t.Errorf("want %#v, got %#v", tt.out, got)
			}
		})
	}
}

func Test_completionTypeIs(t *testing.T) {
	type args struct {
	}
	tests := []struct {
		name            string
		completionTypes []completionType
		expect          completionType
		want            bool
	}{
		{
			completionTypes: []completionType{
				CompletionTypeColumn,
			},
			expect: CompletionTypeColumn,
			want:   true,
		},
		{
			completionTypes: []completionType{
				CompletionTypeTable,
				CompletionTypeView,
				CompletionTypeFunction,
				CompletionTypeColumn,
			},
			expect: CompletionTypeColumn,
			want:   true,
		},
		{
			completionTypes: []completionType{
				CompletionTypeTable,
				CompletionTypeView,
				CompletionTypeFunction,
			},
			expect: CompletionTypeColumn,
			want:   false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := completionTypeIs(tt.completionTypes, tt.expect); got != tt.want {
				t.Errorf("completionTypeIs() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestComplete(t *testing.T) {
	tests := []struct {
		name      string
		text      string
		lowerCase bool
		expected  []lsp.CompletionItem
	}{
		{
			name: "keyword",
			text: "sel",
			expected: []lsp.CompletionItem{
				{
					Label:    "SELECT",
					Kind:     lsp.KeywordCompletion,
					Detail:   "keyword",
					SortText: "9999SELECT",
				},
			},
		},
		{
			name:      "keyword-lowercase",
			text:      "sel",
			lowerCase: true,
			expected: []lsp.CompletionItem{
				{
					Label:    "select",
					Kind:     lsp.KeywordCompletion,
					Detail:   "keyword",
					SortText: "9999select",
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run("", func(t *testing.T) {
			c := NewCompleter(nil)
			got, err := c.Complete("sel", lsp.CompletionParams{
				TextDocumentPositionParams: lsp.TextDocumentPositionParams{
					Position: lsp.Position{
						Line:      0,
						Character: len(tt.text),
					},
				},
			}, tt.lowerCase)
			if err != nil {
				t.Fatal(err)
			}

			if !reflect.DeepEqual(got, tt.expected) {
				t.Errorf("\nwant: %v\ngot:  %v", tt.expected, got)
			}
		})
	}
}

func TestGenerateAlias(t *testing.T) {
	noMatchesTable := make(map[string]interface{})
	noMatchesTable["XX"] = true
	matchesTable := make(map[string]interface{})
	matchesTable["XX"] = true
	matchesTable["T1"] = true

	tests := []struct {
		name  string
		table string
		tMap  map[string]interface{}
		want  string
	}{
		{
			"no matches",
			"Table",
			noMatchesTable,
			"T1",
		},
		{
			"matches",
			"Table",
			matchesTable,
			"T2",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := generateTableAlias(tt.table, tt.tMap); got != tt.want {
				t.Errorf("generateAlias() = %v, want  %v", got, tt.want)
			}
		})
	}
}

func TestComplete_ContextualKeywords(t *testing.T) {
	c := NewCompleter(nil)

	containsLabel := func(items []lsp.CompletionItem, label string, exact bool) bool {
		for _, item := range items {
			if exact {
				if item.Label == label {
					return true
				}
			} else {
				if strings.EqualFold(item.Label, label) {
					return true
				}
			}
		}
		return false
	}

	tests := []struct {
		name          string
		text          string
		cursorLine    int
		cursorCol     int
		lowerCase     bool
		shouldContain []string
		shouldOmit    []string
	}{
		{
			name:          "top-level query start suggests statement starters only",
			text:          "sel",
			cursorLine:    0,
			cursorCol:     3,
			shouldContain: []string{"SELECT"},
			shouldOmit:    []string{"WHERE", "HAVING", "ON", "FROM", "JOIN", "LIMIT"},
		},
		{
			name:          "top-level query start suggests INSERT INTO",
			text:          "ins",
			cursorLine:    0,
			cursorCol:     3,
			shouldContain: []string{"INSERT INTO"},
			shouldOmit:    []string{"WHERE", "JOIN", "FROM", "ON"},
		},
		{
			name:          "top-level query start suggests UPDATE",
			text:          "upd",
			cursorLine:    0,
			cursorCol:     3,
			shouldContain: []string{"UPDATE"},
			shouldOmit:    []string{"WHERE", "JOIN", "FROM", "ON"},
		},
		{
			name:          "top-level query start suggests DELETE FROM",
			text:          "del",
			cursorLine:    0,
			cursorCol:     3,
			shouldContain: []string{"DELETE FROM"},
			shouldOmit:    []string{"WHERE", "JOIN", "FROM", "ON"},
		},
		{
			name:          "after SELECT columns suggests FROM but not WHERE/JOIN",
			text:          "SELECT id, name fr",
			cursorLine:    0,
			cursorCol:     18,
			shouldContain: []string{"FROM"},
			shouldOmit:    []string{"WHERE", "JOIN", "CREATE", "DROP", "ALTER"},
		},
		{
			name:          "in SELECT column list typing wh does NOT suggest WHERE",
			text:          "SELECT id, wh",
			cursorLine:    0,
			cursorCol:     13,
			shouldContain: []string{},
			shouldOmit:    []string{"WHERE"},
		},
		{
			name:          "in SELECT column list typing joi does NOT suggest JOIN",
			text:          "SELECT id, joi",
			cursorLine:    0,
			cursorCol:     14,
			shouldContain: []string{},
			shouldOmit:    []string{"JOIN", "LEFT JOIN", "INNER JOIN"},
		},
		{
			name:          "after FROM table suggests WHERE and JOIN but not SELECT/CREATE",
			text:          "SELECT * FROM users wh",
			cursorLine:    0,
			cursorCol:     22,
			shouldContain: []string{"WHERE"},
			shouldOmit:    []string{"SELECT", "CREATE", "DROP", "ALTER", "FROM"},
		},
		{
			name:          "after FROM table suggests JOIN",
			text:          "SELECT * FROM users joi",
			cursorLine:    0,
			cursorCol:     23,
			shouldContain: []string{"JOIN"},
			shouldOmit:    []string{"SELECT", "INSERT", "CREATE"},
		},
		{
			name:          "after FROM table suggests LEFT JOIN",
			text:          "SELECT * FROM users lef",
			cursorLine:    0,
			cursorCol:     23,
			shouldContain: []string{"LEFT JOIN"},
			shouldOmit:    []string{"SELECT", "INSERT", "CREATE"},
		},
		{
			name:          "after FROM table suggests GROUP BY",
			text:          "SELECT * FROM users gro",
			cursorLine:    0,
			cursorCol:     23,
			shouldContain: []string{"GROUP BY"},
			shouldOmit:    []string{"SELECT", "INSERT", "CREATE"},
		},
		{
			name:          "after FROM table suggests ORDER BY",
			text:          "SELECT * FROM users ord",
			cursorLine:    0,
			cursorCol:     23,
			shouldContain: []string{"ORDER BY"},
			shouldOmit:    []string{"SELECT", "INSERT", "CREATE"},
		},
		{
			name:          "after FROM table suggests LIMIT",
			text:          "SELECT * FROM users lim",
			cursorLine:    0,
			cursorCol:     23,
			shouldContain: []string{"LIMIT"},
			shouldOmit:    []string{"SELECT", "INSERT", "CREATE"},
		},
		{
			name:          "after JOIN table suggests ON and USING",
			text:          "SELECT * FROM users JOIN orders o",
			cursorLine:    0,
			cursorCol:     33,
			shouldContain: []string{"ON"},
			shouldOmit:    []string{"FROM", "SELECT", "INSERT", "WHERE"},
		},
		{
			name:          "after WHERE condition suggests logical operators and clauses",
			text:          "SELECT * FROM users WHERE id = 1 an",
			cursorLine:    0,
			cursorCol:     35,
			shouldContain: []string{"AND"},
			shouldOmit:    []string{"FROM", "CREATE", "DROP", "JOIN"},
		},
		{
			name:          "after WHERE condition suggests OR",
			text:          "SELECT * FROM users WHERE id = 1 o",
			cursorLine:    0,
			cursorCol:     34,
			shouldContain: []string{"OR", "ORDER BY"},
			shouldOmit:    []string{"FROM", "SELECT", "JOIN"},
		},
		{
			name:          "after ORDER BY column suggests ASC and DESC",
			text:          "SELECT * FROM users ORDER BY name de",
			cursorLine:    0,
			cursorCol:     36,
			shouldContain: []string{"DESC"},
			shouldOmit:    []string{"FROM", "WHERE", "JOIN", "SELECT"},
		},
		{
			name:          "after ORDER BY column suggests ASC",
			text:          "SELECT * FROM users ORDER BY name as",
			cursorLine:    0,
			cursorCol:     36,
			shouldContain: []string{"ASC"},
			shouldOmit:    []string{"FROM", "WHERE", "JOIN", "SELECT"},
		},
		{
			name:          "after GROUP BY column suggests HAVING",
			text:          "SELECT department, count(*) FROM employees GROUP BY department hav",
			cursorLine:    0,
			cursorCol:     66,
			shouldContain: []string{"HAVING"},
			shouldOmit:    []string{"FROM", "WHERE", "JOIN", "SELECT"},
		},
		{
			name:          "UPDATE statement suggests SET",
			text:          "UPDATE users se",
			cursorLine:    0,
			cursorCol:     15,
			shouldContain: []string{"SET"},
			shouldOmit:    []string{"SELECT", "FROM", "GROUP BY"},
		},
		{
			name:          "DELETE statement suggests WHERE",
			text:          "DELETE FROM users wh",
			cursorLine:    0,
			cursorCol:     20,
			shouldContain: []string{"WHERE"},
			shouldOmit:    []string{"SELECT", "FROM", "JOIN", "GROUP BY"},
		},
		{
			name:          "INSERT statement suggests VALUES",
			text:          "INSERT INTO users (id, name) val",
			cursorLine:    0,
			cursorCol:     32,
			shouldContain: []string{"VALUES"},
			shouldOmit:    []string{"WHERE", "JOIN", "GROUP BY"},
		},
		{
			name:          "newline recovery without semicolon maintains top-level statement starter",
			text:          "SELECT * FROM users\nsel",
			cursorLine:    1,
			cursorCol:     3,
			shouldContain: []string{"SELECT"},
			shouldOmit:    []string{"WHERE", "FROM", "JOIN", "HAVING"},
		},
		{
			name:          "newline recovery without semicolon after INSERT",
			text:          "INSERT INTO users (id) VALUES (1)\nup",
			cursorLine:    1,
			cursorCol:     2,
			shouldContain: []string{"UPDATE"},
			shouldOmit:    []string{"WHERE", "FROM", "JOIN"},
		},
		{
			name:          "newline after semicolon maintains top-level statement starter",
			text:          "SELECT * FROM users;\nsel",
			cursorLine:    1,
			cursorCol:     3,
			shouldContain: []string{"SELECT"},
			shouldOmit:    []string{"WHERE", "FROM", "JOIN", "HAVING"},
		},
		{
			name:          "subquery enclosed in parens suggests inner query keywords",
			text:          "SELECT * FROM (SELECT id, name fr)",
			cursorLine:    0,
			cursorCol:     33,
			shouldContain: []string{"FROM"},
			shouldOmit:    []string{"WHERE", "JOIN", "CREATE", "DROP"},
		},
		{
			name:          "lowercase keywords preference respected",
			text:          "sel",
			cursorLine:    0,
			cursorCol:     3,
			lowerCase:     true,
			shouldContain: []string{"select"},
			shouldOmit:    []string{"SELECT", "WHERE"},
		},
		{
			name:          "member identifier access does not suggest SQL keywords",
			text:          "SELECT u.",
			cursorLine:    0,
			cursorCol:     9,
			shouldContain: []string{},
			shouldOmit:    []string{"SELECT", "WHERE", "FROM", "JOIN", "GROUP BY"},
		},
		{
			name:          "ClickHouse FORMAT keyword after table reference",
			text:          "SELECT * FROM users for",
			cursorLine:    0,
			cursorCol:     23,
			shouldContain: []string{"FORMAT"},
			shouldOmit:    []string{"SELECT", "INSERT"},
		},
		{
			name:          "ClickHouse FINAL keyword after table reference",
			text:          "SELECT * FROM users fi",
			cursorLine:    0,
			cursorCol:     22,
			shouldContain: []string{"FINAL"},
			shouldOmit:    []string{"SELECT", "INSERT"},
		},
		{
			name:          "ClickHouse PREWHERE keyword after table reference",
			text:          "SELECT * FROM users pre",
			cursorLine:    0,
			cursorCol:     23,
			shouldContain: []string{"PREWHERE"},
			shouldOmit:    []string{"SELECT", "INSERT"},
		},
		{
			name:          "cursor at col 0 only suggests statement starters, never middle keywords like ABORT/AND/WHERE",
			text:          "\n",
			cursorLine:    1,
			cursorCol:     0,
			shouldContain: []string{"SELECT", "INSERT INTO", "UPDATE", "DELETE FROM", "WITH", "CREATE TABLE"},
			shouldOmit:    []string{"WHERE", "HAVING", "ON", "FROM", "JOIN", "ABORT", "ACTION", "ADD", "AND", "AS", "ASC", "WINDOW"},
		},
		{
			name:          "between SELECT and FROM does not suggest FROM or statement starters",
			text:          "SELECT  FROM users;",
			cursorLine:    0,
			cursorCol:     7,
			shouldContain: []string{"DISTINCT"},
			shouldOmit:    []string{"FROM", "WHERE", "JOIN", "SELECT", "INSERT INTO", "CREATE TABLE"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			items, err := c.Complete(tt.text, lsp.CompletionParams{
				TextDocumentPositionParams: lsp.TextDocumentPositionParams{
					Position: lsp.Position{
						Line:      tt.cursorLine,
						Character: tt.cursorCol,
					},
				},
			}, tt.lowerCase)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			for _, expected := range tt.shouldContain {
				if !containsLabel(items, expected, tt.lowerCase) {
					t.Errorf("expected completion to contain %q, but was missing. Items: %v", expected, items)
				}
			}

			for _, omitted := range tt.shouldOmit {
				if containsLabel(items, omitted, tt.lowerCase) {
					t.Errorf("expected completion to OMIT %q in this context, but it was present", omitted)
				}
			}
		})
	}
}
