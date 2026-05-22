package cache_test

import (
	"testing"

	"github.com/goccy/go-json"

	"tgp/internal/model"
	"tgp/internal/tags"
	"tgp/plugins/astg/cache"
)

func TestMarshalProjectAfterSanitize(t *testing.T) {

	project := &model.Project{
		Version: "test",
		Types:   make(map[string]*model.Type),
		Contracts: []*model.Contract{
			{
				Name: "C",
				Implementations: []*model.ImplementationInfo{
					{
						MethodsMap: map[string]*model.ImplementationMethod{
							"m":  nil,
							"ok": {Name: "ok", ErrorTypes: []*model.ErrorTypeReference{nil, {FullName: "e"}}},
						},
					},
					nil,
				},
				Methods: []*model.Method{nil, {Name: "M", Annotations: tags.DocTags{"x": "y"}}},
			},
			nil,
		},
	}

	removed := cache.SanitizeProject(project)
	if removed == 0 {
		t.Fatalf("expected removed nil entries")
	}

	if len(project.Contracts) != 1 {
		t.Fatalf("contracts: got %d want 1", len(project.Contracts))
	}

	if _, err := json.Marshal(project); err != nil {
		t.Fatalf("marshal: %v", err)
	}
}
