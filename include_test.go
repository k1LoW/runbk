package runn

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/k1LoW/runn/testutil"
)

func TestIncludeRunnerRun(t *testing.T) {
	tests := []struct {
		path string
		vars map[string]any
		want int
	}{
		{"testdata/book/db.yml", map[string]any{}, 8},
		{"testdata/book/db.yml", map[string]any{"foo": "bar"}, 8},
		{"testdata/book/db.yml", map[string]any{"json": "json://../vars.json"}, 8},
		{"https://raw.githubusercontent.com/k1LoW/runn/main/testdata/book/db.yml", map[string]any{}, 8},
	}
	ctx := context.Background()
	for _, tt := range tests {
		_, dsn := testutil.SQLite(t)
		o, err := New(Runner("db", dsn), Scopes(ScopeAllowReadRemote))
		if err != nil {
			t.Fatal(err)
		}
		r, err := newIncludeRunner()
		if err != nil {
			t.Fatal(err)
		}
		s := newStep(0, "stepKey", o, nil)
		s.includeConfig = &includeConfig{path: tt.path, vars: tt.vars}
		if err := r.Run(ctx, s); err != nil {
			t.Fatal(err)
		}

		t.Run("step length", func(t *testing.T) {
			{
				got := len(o.store.steps)
				if want := 1; got != want {
					t.Errorf("got %v\nwant %v", got, want)
				}
			}
			{
				got := len(o.store.steps[0]["steps"].([]map[string]any))
				if got != tt.want {
					t.Errorf("got %v\nwant %v", got, tt.want)
				}
			}
		})

		t.Run("var length", func(t *testing.T) {
			{
				got := len(o.store.vars)
				if want := 0; got != want {
					t.Errorf("got %v\nwant %v", got, want)
				}
			}
			{
				got := len(o.store.steps[0]["vars"].(map[string]any))
				if want := len(tt.vars); got != want {
					t.Errorf("got %v\nwant %v", got, want)
				}
			}
		})
	}
}

func TestCustomRunner(t *testing.T) {
	tests := []struct {
		path string
	}{
		{"testdata/book/custom_runners.yml"},
	}
	ctx := context.Background()
	for _, tt := range tests {
		ts := testutil.HTTPServer(t)
		t.Setenv("TEST_HTTP_ENDPOINT", ts.URL)
		o, err := New(Book(tt.path))
		if err != nil {
			t.Fatal(err)
		}
		if err := o.Run(ctx); err != nil {
			t.Fatal(err)
		}
	}
}

func TestMultipleIncludeRunnerRun(t *testing.T) {
	tests := []struct {
		path string
		vars map[string]any
	}{
		{
			"testdata/book/multiple_include_a.yml",
			map[string]any{
				"foo":  123,
				"bar":  "123-abc",
				"baz":  "-23",
				"qux":  4,
				"quxx": "2",
				"corge": map[string]any{
					"grault": "1234",
					"garply": 1234,
				},
				"waldo": true,
				"fred":  "false",
			},
		},
		{
			"testdata/book/multiple_include_main.yml",
			map[string]any{
				"foo":  123,
				"bar":  "abc",
				"baz":  100,
				"qux":  -1,
				"quxx": 2,
				"corge": map[string]any{
					"grault": "1234",
					"garply": 1234,
				},
				"waldo": true,
				"fred":  "false",
			},
		},
	}
	ctx := context.Background()
	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			o, err := New(Runner("req", "https://example.com"))
			if err != nil {
				t.Fatal(err)
			}
			r, err := newIncludeRunner()
			if err != nil {
				t.Fatal(err)
			}
			s := newStep(0, "stepKey", o, nil)
			s.includeConfig = &includeConfig{path: tt.path, vars: tt.vars}
			if err := r.Run(ctx, s); err != nil {
				t.Error(err)
			}
		})
	}
}

func TestUseParentStore(t *testing.T) {
	host := testutil.CreateHTTPBinContainer(t)
	tests := []struct {
		name        string
		path        string
		parentStore store
		wantErr     bool
	}{
		{
			"Use parent store in vars: section",
			"testdata/book/use_parent_store_vars.yml",
			store{
				vars: map[string]any{
					"foo": "bar",
				},
			},
			false,
		},
		{
			"Error if there is no parent store",
			"testdata/book/use_parent_store_vars.yml",
			store{},
			true,
		},
		{
			"Use parent store in runners: section",
			"testdata/book/use_parent_store_runners.yml",
			store{
				vars: map[string]any{
					"httprunner": host,
				},
			},
			false,
		},
	}
	ctx := context.Background()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			o, err := New(Runner("req", "https://example.com"))
			if err != nil {
				t.Fatal(err)
			}
			o.store = &tt.parentStore
			r, err := newIncludeRunner()
			if err != nil {
				t.Fatal(err)
			}
			s := newStep(0, "stepKey", o, nil)
			s.includeConfig = &includeConfig{path: tt.path}
			if err := r.Run(ctx, s); err != nil {
				if !tt.wantErr {
					t.Error(err)
				}
				return
			}
			if tt.wantErr {
				t.Error("want error")
			}
		})
	}
}

func TestIncludeVars(t *testing.T) {
	tests := []struct {
		path string
	}{
		{"testdata/book/parent_vars_parent.yml"},
		{"testdata/book/include_vars_main.yml"},
	}
	ctx := context.Background()
	for _, tt := range tests {
		o, err := New(Book(tt.path))
		if err != nil {
			t.Fatal(err)
		}
		if err := o.Run(ctx); err != nil {
			t.Fatal(err)
		}
	}
}

func TestIncludedRunErr(t *testing.T) {
	dummyErr := errors.New("dummy")
	tests := []struct {
		target error
		want   bool
	}{
		{errors.New("dummy"), false},
		{&includedRunErr{err: dummyErr}, true},
		{fmt.Errorf("dummy: %w", &includedRunErr{err: dummyErr}), true},
		{fmt.Errorf("dummy: %w", fmt.Errorf("dummy: %w", &includedRunErr{err: dummyErr})), true},
		{fmt.Errorf("dummy: %w", fmt.Errorf("dummy: %w", dummyErr)), false},
	}
	for _, tt := range tests {
		if got := errors.Is(&includedRunErr{err: dummyErr}, tt.target); got != tt.want {
			t.Errorf("got %v\nwant %v", got, tt.want)
		}
	}
}
