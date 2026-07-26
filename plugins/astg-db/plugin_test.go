package main

import (
	"errors"
	"path/filepath"
	"testing"

	"tgp/core"
	"tgp/core/data"
	"tgp/internal/cdb"
	"tgp/internal/model"
)

func TestExecuteNoop(t *testing.T) {

	t.Parallel()

	plugin := &AstgDbPlugin{}

	t.Run("nil request", func(t *testing.T) {
		t.Parallel()
		response, err := plugin.Execute(nil)
		if err != nil {
			t.Fatalf("Execute: %v", err)
		}
		if response != nil {
			t.Fatalf("expected nil response")
		}
	})

	t.Run("without from-db", func(t *testing.T) {
		t.Parallel()
		request := data.NewStorage()
		response, err := plugin.Execute(request)
		if err != nil {
			t.Fatalf("Execute: %v", err)
		}
		if response.Has("project") {
			t.Fatal("project must not be set")
		}
	})

	t.Run("with existing project", func(t *testing.T) {
		t.Parallel()
		request := data.NewStorage()
		if err := request.Set("project", &model.Project{ModulePath: "example.com/app"}); err != nil {
			t.Fatalf("Set project: %v", err)
		}
		if err := request.Set(optionFromDB, "ignored@main"); err != nil {
			t.Fatalf("Set from-db: %v", err)
		}
		response, err := plugin.Execute(request)
		if err != nil {
			t.Fatalf("Execute: %v", err)
		}
		project, getErr := data.Get[*model.Project](response, "project")
		if getErr != nil {
			t.Fatalf("Get project: %v", getErr)
		}
		if project.ModulePath != "example.com/app" {
			t.Fatalf("project must stay unchanged, got %q", project.ModulePath)
		}
	})
}

func TestLoadFromDB(t *testing.T) {

	// Не параллелим: stub InteractiveSelect меняет пакетную переменную core.InteractiveSelect.
	const (
		projectKey = "example.com.app"
		version    = "release-stage"
	)

	root := t.TempDir()
	project := &model.Project{
		ModulePath: "example.com/app",
		Contracts: []*model.Contract{
			{Name: "Alpha", ID: "alpha"},
			{Name: "Beta", ID: "beta"},
			{Name: "Gamma", ID: "gamma"},
		},
	}
	seedContractsDB(t, root, projectKey, version, project)

	t.Run("ref without contracts selects interactively", func(t *testing.T) {
		restore := stubInteractiveSelect(t, func(prompt string, options []string, multiSelect bool, _ []string) ([]string, error) {
			if !multiSelect {
				t.Fatalf("expected multi-select for contracts, prompt=%q", prompt)
			}
			if len(options) != 3 {
				t.Fatalf("options: got %v", options)
			}
			return []string{"Alpha", "Gamma"}, nil
		})
		defer restore()

		request := data.NewStorage()
		if err := request.Set(optionFromDB, projectKey+"@"+version); err != nil {
			t.Fatalf("Set from-db: %v", err)
		}

		response, err := loadFromDB(request, root)
		if err != nil {
			t.Fatalf("loadFromDB: %v", err)
		}
		assertContractNames(t, response, "Alpha", "Gamma")
	})

	t.Run("empty from-db selects ref then contracts", func(t *testing.T) {
		step := 0
		restore := stubInteractiveSelect(t, func(prompt string, options []string, multiSelect bool, _ []string) ([]string, error) {
			step++
			switch step {
			case 1:
				if multiSelect {
					t.Fatalf("ref select must be single, prompt=%q", prompt)
				}
				wantRef := projectKey + "@" + version
				for _, option := range options {
					if option == wantRef {
						return []string{wantRef}, nil
					}
				}
				t.Fatalf("ref %q not in options %v", wantRef, options)
				return nil, nil
			case 2:
				if !multiSelect {
					t.Fatalf("contracts select must be multi, prompt=%q", prompt)
				}
				return []string{"Beta"}, nil
			default:
				t.Fatalf("unexpected InteractiveSelect call #%d prompt=%q", step, prompt)
				return nil, nil
			}
		})
		defer restore()

		request := data.NewStorage()
		if err := request.Set(optionFromDB, ""); err != nil {
			t.Fatalf("Set from-db: %v", err)
		}

		response, err := loadFromDB(request, root)
		if err != nil {
			t.Fatalf("loadFromDB: %v", err)
		}
		if step != 2 {
			t.Fatalf("InteractiveSelect calls: got %d want 2", step)
		}
		assertContractNames(t, response, "Beta")
	})

	t.Run("contracts in ref skip interactive select", func(t *testing.T) {
		restore := stubInteractiveSelect(t, func(string, []string, bool, []string) ([]string, error) {
			t.Fatal("InteractiveSelect must not be called")
			return nil, nil
		})
		defer restore()

		request := data.NewStorage()
		if err := request.Set(optionFromDB, projectKey+":Alpha,Beta@"+version); err != nil {
			t.Fatalf("Set from-db: %v", err)
		}

		response, err := loadFromDB(request, root)
		if err != nil {
			t.Fatalf("loadFromDB: %v", err)
		}
		assertContractNames(t, response, "Alpha", "Beta")
	})

	t.Run("contracts option skips interactive select", func(t *testing.T) {
		restore := stubInteractiveSelect(t, func(string, []string, bool, []string) ([]string, error) {
			t.Fatal("InteractiveSelect must not be called")
			return nil, nil
		})
		defer restore()

		request := data.NewStorage()
		if err := request.Set(optionFromDB, projectKey+"@"+version); err != nil {
			t.Fatalf("Set from-db: %v", err)
		}
		if err := request.Set("contracts", "Gamma"); err != nil {
			t.Fatalf("Set contracts: %v", err)
		}

		response, err := loadFromDB(request, root)
		if err != nil {
			t.Fatalf("loadFromDB: %v", err)
		}
		// Фильтр --contracts применяется в command-плагине; astg-db оставляет все контракты.
		assertContractNames(t, response, "Alpha", "Beta", "Gamma")
	})

	t.Run("all-contracts skips interactive select", func(t *testing.T) {
		restore := stubInteractiveSelect(t, func(string, []string, bool, []string) ([]string, error) {
			t.Fatal("InteractiveSelect must not be called")
			return nil, nil
		})
		defer restore()

		request := data.NewStorage()
		if err := request.Set(optionFromDB, projectKey+"@"+version); err != nil {
			t.Fatalf("Set from-db: %v", err)
		}
		if err := request.Set(optionAllContracts, true); err != nil {
			t.Fatalf("Set all-contracts: %v", err)
		}

		response, err := loadFromDB(request, root)
		if err != nil {
			t.Fatalf("loadFromDB: %v", err)
		}
		assertContractNames(t, response, "Alpha", "Beta", "Gamma")
	})

	t.Run("empty contracts selection fails", func(t *testing.T) {
		restore := stubInteractiveSelect(t, func(string, []string, bool, []string) ([]string, error) {
			return []string{}, nil
		})
		defer restore()

		request := data.NewStorage()
		if err := request.Set(optionFromDB, projectKey+"@"+version); err != nil {
			t.Fatalf("Set from-db: %v", err)
		}

		_, err := loadFromDB(request, root)
		if err == nil {
			t.Fatal("expected error")
		}
	})

	t.Run("interactive select error", func(t *testing.T) {
		restore := stubInteractiveSelect(t, func(string, []string, bool, []string) ([]string, error) {
			return nil, errors.New("cancelled")
		})
		defer restore()

		request := data.NewStorage()
		if err := request.Set(optionFromDB, projectKey+"@"+version); err != nil {
			t.Fatalf("Set from-db: %v", err)
		}

		_, err := loadFromDB(request, root)
		if err == nil {
			t.Fatal("expected error")
		}
	})

	t.Run("unknown project", func(t *testing.T) {
		request := data.NewStorage()
		if err := request.Set(optionFromDB, "missing.project@main"); err != nil {
			t.Fatalf("Set from-db: %v", err)
		}

		_, err := loadFromDB(request, root)
		if err == nil {
			t.Fatal("expected error")
		}
	})

	t.Run("invalid ref", func(t *testing.T) {
		request := data.NewStorage()
		if err := request.Set(optionFromDB, "@only-version"); err != nil {
			t.Fatalf("Set from-db: %v", err)
		}

		_, err := loadFromDB(request, root)
		if err == nil {
			t.Fatal("expected error")
		}
	})
}

func seedContractsDB(t *testing.T, root string, projectKey string, version string, project *model.Project) {

	t.Helper()

	idx := &cdb.Index{
		Version:  1,
		Aliases:  map[string]string{},
		Projects: map[string]cdb.ProjectMeta{},
	}

	projectFile, err := cdb.UpsertProject(root, idx, projectKey, projectKey, project.ModulePath, version, cdb.VersionKindBranch)
	if err != nil {
		t.Fatalf("UpsertProject: %v", err)
	}
	if err = cdb.SaveIndex(root, idx); err != nil {
		t.Fatalf("SaveIndex: %v", err)
	}
	if err = cdb.WriteProject(root, projectFile, project); err != nil {
		t.Fatalf("WriteProject: %v", err)
	}
	if _, err = cdb.ReadProject(root, filepath.Clean(projectFile)); err != nil {
		t.Fatalf("ReadProject smoke: %v", err)
	}
}

func stubInteractiveSelect(t *testing.T, fn func(prompt string, options []string, multiSelect bool, defaultOptions []string) ([]string, error)) (restore func()) {

	t.Helper()

	previous := core.InteractiveSelect
	core.InteractiveSelect = fn
	return func() {
		core.InteractiveSelect = previous
	}
}

func assertContractNames(t *testing.T, response data.Storage, want ...string) {

	t.Helper()

	project, err := data.Get[*model.Project](response, "project")
	if err != nil {
		t.Fatalf("Get project: %v", err)
	}
	if len(project.Contracts) != len(want) {
		names := make([]string, 0, len(project.Contracts))
		for _, contract := range project.Contracts {
			names = append(names, contract.Name)
		}
		t.Fatalf("contracts: got %v want %v", names, want)
	}
	for i, name := range want {
		if project.Contracts[i].Name != name {
			t.Fatalf("contracts[%d]: got %q want %q", i, project.Contracts[i].Name, name)
		}
	}
}
