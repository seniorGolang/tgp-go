package helper

import (
	"testing"

	"tgp/core/data"
	"tgp/internal/model"
)

func testProject() *model.Project {

	return &model.Project{
		ModulePath: "example.com/app",
		Contracts: []*model.Contract{
			{ID: "example.com/app/contracts:Alpha", Name: "Alpha"},
			{ID: "example.com/app/contracts:Beta", Name: "Beta"},
			{ID: "example.com/app/contracts:Hooks", Name: "Hooks"},
		},
		Types: map[string]*model.Type{"t1": {TypeName: "T1"}},
	}
}

func contractNames(project *model.Project) []string {

	names := make([]string, 0, len(project.Contracts))
	for _, c := range project.Contracts {
		names = append(names, c.Name)
	}
	return names
}

func TestFilterContractsByInterfaces(t *testing.T) {

	t.Parallel()
	project := testProject()

	t.Run("empty keeps all", func(t *testing.T) {
		t.Parallel()
		got, err := FilterContractsByInterfaces(project, nil)
		if err != nil {
			t.Fatalf("err: %v", err)
		}
		if len(got) != 3 {
			t.Fatalf("got %d contracts, want 3", len(got))
		}
	})

	t.Run("include by name", func(t *testing.T) {
		t.Parallel()
		got, err := FilterContractsByInterfaces(project, []string{"Alpha", "Hooks"})
		if err != nil {
			t.Fatalf("err: %v", err)
		}
		if len(got) != 2 || got[0].Name != "Alpha" || got[1].Name != "Hooks" {
			t.Fatalf("got %#v", contractNames(&model.Project{Contracts: got}))
		}
	})

	t.Run("include by id", func(t *testing.T) {
		t.Parallel()
		got, err := FilterContractsByInterfaces(project, []string{"example.com/app/contracts:Beta"})
		if err != nil {
			t.Fatalf("err: %v", err)
		}
		if len(got) != 1 || got[0].Name != "Beta" {
			t.Fatalf("got %#v", contractNames(&model.Project{Contracts: got}))
		}
	})

	t.Run("exclude", func(t *testing.T) {
		t.Parallel()
		got, err := FilterContractsByInterfaces(project, []string{"!Hooks"})
		if err != nil {
			t.Fatalf("err: %v", err)
		}
		if len(got) != 2 || got[0].Name != "Alpha" || got[1].Name != "Beta" {
			t.Fatalf("got %#v", contractNames(&model.Project{Contracts: got}))
		}
	})

	t.Run("exclude by id", func(t *testing.T) {
		t.Parallel()
		got, err := FilterContractsByInterfaces(project, []string{"!example.com/app/contracts:Hooks"})
		if err != nil {
			t.Fatalf("err: %v", err)
		}
		if len(got) != 2 || got[0].Name != "Alpha" || got[1].Name != "Beta" {
			t.Fatalf("got %#v", contractNames(&model.Project{Contracts: got}))
		}
	})

	t.Run("include and exclude conflict", func(t *testing.T) {
		t.Parallel()
		_, err := FilterContractsByInterfaces(project, []string{"Alpha", "!Hooks"})
		if err == nil {
			t.Fatal("expected error")
		}
	})

	t.Run("unknown include yields empty", func(t *testing.T) {
		t.Parallel()
		got, err := FilterContractsByInterfaces(project, []string{"Missing"})
		if err != nil {
			t.Fatalf("err: %v", err)
		}
		if len(got) != 0 {
			t.Fatalf("got %d, want 0", len(got))
		}
	})
}

func TestProjectWithFilter(t *testing.T) {

	t.Parallel()

	t.Run("empty returns same pointer", func(t *testing.T) {
		t.Parallel()
		project := testProject()
		out, err := ProjectWithFilter(project, nil)
		if err != nil {
			t.Fatalf("err: %v", err)
		}
		if out != project {
			t.Fatal("expected same project pointer")
		}
	})

	t.Run("filter does not mutate original", func(t *testing.T) {
		t.Parallel()
		project := testProject()
		out, err := ProjectWithFilter(project, []string{"!Hooks"})
		if err != nil {
			t.Fatalf("err: %v", err)
		}
		if out == project {
			t.Fatal("expected shallow copy")
		}
		if len(project.Contracts) != 3 {
			t.Fatalf("original mutated: %d contracts", len(project.Contracts))
		}
		if len(out.Contracts) != 2 {
			t.Fatalf("filtered: got %d, want 2", len(out.Contracts))
		}
		if out.Types == nil || out.Types["t1"] == nil {
			t.Fatal("types must be preserved on copy")
		}
		if out.ModulePath != project.ModulePath {
			t.Fatalf("module path: %q", out.ModulePath)
		}
	})

	t.Run("include only Hooks", func(t *testing.T) {
		t.Parallel()
		project := testProject()
		out, err := ProjectWithFilter(project, []string{"Hooks"})
		if err != nil {
			t.Fatalf("err: %v", err)
		}
		if len(out.Contracts) != 1 || out.Contracts[0].Name != "Hooks" {
			t.Fatalf("got %#v", contractNames(out))
		}
		if len(project.Contracts) != 3 {
			t.Fatal("original must stay full")
		}
	})

	t.Run("conflict", func(t *testing.T) {
		t.Parallel()
		project := testProject()
		_, err := ProjectWithFilter(project, []string{"Alpha", "!Beta"})
		if err == nil {
			t.Fatal("expected error")
		}
	})
}

func TestProjectWithFilterPipelineInvariant(t *testing.T) {

	t.Parallel()

	storage := data.NewStorage()
	full := testProject()
	if err := storage.Set("project", full); err != nil {
		t.Fatalf("Set project: %v", err)
	}

	stored, err := GetProject(storage)
	if err != nil {
		t.Fatalf("GetProject: %v", err)
	}

	var filtered *model.Project
	if filtered, err = ProjectWithFilter(stored, []string{"!Hooks"}); err != nil {
		t.Fatalf("filter !Hooks: %v", err)
	}
	if len(filtered.Contracts) != 2 {
		t.Fatalf("filtered !Hooks: got %d", len(filtered.Contracts))
	}

	if filtered, err = ProjectWithFilter(stored, []string{"Hooks"}); err != nil {
		t.Fatalf("filter Hooks: %v", err)
	}
	if len(filtered.Contracts) != 1 || filtered.Contracts[0].Name != "Hooks" {
		t.Fatalf("filtered Hooks: %#v", contractNames(filtered))
	}

	again, err := GetProject(storage)
	if err != nil {
		t.Fatalf("GetProject after filters: %v", err)
	}
	if len(again.Contracts) != 3 {
		t.Fatalf("storage contracts: got %d want 3 (%v)", len(again.Contracts), contractNames(again))
	}
	if len(full.Contracts) != 3 {
		t.Fatalf("original contracts: got %d want 3", len(full.Contracts))
	}
}
