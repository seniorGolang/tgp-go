package cache_test

import (
	"testing"

	"github.com/goccy/go-json"

	"tgp/internal/model"
	"tgp/plugins/astg/cache"
)

func variadicMapArgProject() *model.Project {

	return &model.Project{
		Version: "test",
		Contracts: []*model.Contract{
			{
				Name: "Rule",
				Methods: []*model.Method{
					{
						Name: "Call",
						Args: []*model.Variable{
							{Name: "ctx", TypeRef: model.TypeRef{TypeID: "context:Context"}},
							{
								Name:       "additional",
								TypeRef: model.TypeRef{
									IsSlice:    true,
									IsEllipsis: true,
									MapKey:     &model.TypeRef{TypeID: "string"},
									MapValue:   &model.TypeRef{TypeID: "any"},
								},
							},
						},
					},
				},
			},
		},
	}
}

func TestMarshalVariadicMapArgWithoutTypeID(t *testing.T) {

	project := variadicMapArgProject()

	if _, err := json.Marshal(project); err != nil {
		t.Fatalf("goccy marshal before sanitize: %v", err)
	}
}

func TestMarshalVariadicMapArgAfterSanitize(t *testing.T) {

	project := variadicMapArgProject()
	cache.SanitizeProject(project)

	if _, err := json.Marshal(project); err != nil {
		t.Fatalf("goccy marshal after sanitize: %v", err)
	}
}

func TestUnmarshalVariadicMapArgJSONThenGoccyMarshal(t *testing.T) {

	raw := []byte(`{
		"version":"test",
		"contracts":[{
			"name":"Rule",
			"methods":[{
				"name":"Call",
				"args":[
					{"typeID":"context:Context","name":"ctx"},
					{"isSlice":true,"isEllipsis":true,"mapKey":{"typeID":"string"},"mapValue":{"typeID":"any"},"name":"additional"}
				]
			}]
		}]
	}`)

	var project model.Project

	if err := json.Unmarshal(raw, &project); err != nil {
		t.Fatalf("goccy unmarshal: %v", err)
	}

	cache.SanitizeProject(&project)

	if _, err := json.Marshal(&project); err != nil {
		t.Fatalf("goccy marshal after roundtrip: %v", err)
	}
}
