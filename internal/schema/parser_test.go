package schema

import (
	"testing"

	"github.com/ory/x/snapshotx"
	"github.com/stretchr/testify/assert"

	"github.com/ory/keto/internal/namespace/ast"
)

var parserErrorTestCases = []struct{ name, input string }{
	{"lexer error", "/* unclosed comment"},
}

var parserTestCases = []struct {
	name, input string
}{
	{"full example", `
  import { Namespace, SubjectSet, FooBar, Anything } from '@ory/keto-namespace-types'

  class User implements Namespace {
	related: {
	  manager: User[]
	}
  }
  
  class Group implements Namespace {
	related: {
	  members: (User | Group)[]
	}
  }
  
  class Folder implements Namespace {
	related: {
	  parents: File[]
	  viewers: SubjectSet<Group, "members">[]
	}
  
	permits = {
	  view: (ctx: Context): boolean => this.related.viewers.includes(ctx.subject),
	}
  }
  
  class File implements Namespace {
	related: {
	  parents: (File | Folder)[]
	  viewers: (User | SubjectSet<Group, "members">)[]
	  owners: (User | SubjectSet<Group, "members">)[]
	  siblings: File[]
	}
  
	// Some comment
	permits = {
	  view: (ctx: Context): boolean =>
	    (
		this.related.parents.traverse((p) =>
		  p.related.viewers.includes(ctx.subject),
		) &&
		this.related.parents.traverse(p => p.permits.view(ctx)) ) ||
		(this.related.viewers.includes(ctx.subject) ||
		this.related.viewers.includes(ctx.subject) ||
		this.related.viewers.includes(ctx.subject) ) ||
		this.related.owners.includes(ctx.subject),
  
	  edit: (ctx: Context) => this.related.owners.includes(ctx.subject),

	  not: (ctx: Context) => !this.related.owners.includes(ctx.subject),
  
	  rename: (ctx: Context) =>
		this.related.siblings.traverse(s => s.permits.edit(ctx)),
	}
  }
`},
}

func TestParser(t *testing.T) {
	t.Run("suite=snapshots", func(t *testing.T) {
		for _, tc := range parserTestCases {
			t.Run(tc.name, func(t *testing.T) {
				ns, errs := Parse(tc.input)
				if len(errs) > 0 {
					for _, err := range errs {
						t.Error(err)
					}
				}
				t.Logf("namespaces:\n%+v", ns)
				nsMap := make(map[string][]ast.Relation)
				for _, n := range ns {
					nsMap[n.Name] = n.Relations
				}
				snapshotx.SnapshotT(t, nsMap)
			})
		}
	})

	t.Run("suite=errors", func(t *testing.T) {
		for _, tc := range parserErrorTestCases {
			t.Run(tc.name, func(t *testing.T) {
				_, errs := Parse(tc.input)
				if len(errs) == 0 {
					t.Error("expected error, but got none")
				}
			})
		}
	})
}

func FuzzParser(f *testing.F) {
	for _, tc := range lexableTestCases {
		f.Add(tc.input)
	}
	for _, tc := range parserTestCases {
		f.Add(tc.input)
	}

	f.Fuzz(func(_ *testing.T, input string) {
		Parse(input)
	})
}

func Test_simplify(t *testing.T) {
	testCases := []struct {
		name            string
		input, expected *ast.SubjectSetRewrite
	}{
		{"empty", nil, nil},
		{
			name: "merge all unions",
			input: &ast.SubjectSetRewrite{
				Operation: ast.OperatorOr,
				Children: ast.Children{
					&ast.ComputedSubjectSet{Relation: "A"},
					&ast.SubjectSetRewrite{
						Children: ast.Children{
							&ast.ComputedSubjectSet{Relation: "B"},
							&ast.SubjectSetRewrite{
								Children: ast.Children{
									&ast.ComputedSubjectSet{Relation: "C"},
									&ast.SubjectSetRewrite{
										Children: ast.Children{
											&ast.ComputedSubjectSet{Relation: "D"},
										},
									},
								},
							},
						},
					},
				},
			},
			expected: &ast.SubjectSetRewrite{
				Children: ast.Children{
					&ast.ComputedSubjectSet{Relation: "A"},
					&ast.ComputedSubjectSet{Relation: "B"},
					&ast.ComputedSubjectSet{Relation: "C"},
					&ast.ComputedSubjectSet{Relation: "D"},
				},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.expected, simplifyExpression(tc.input))
		})
	}
}
