package data_test

import (
	"testing"

	"tgp/core/data"
	"tgp/internal/model"
)

func TestMarshalValueProjectWithNilEntries(t *testing.T) {

	project := &model.Project{
		Contracts: []*model.Contract{{
			Methods: []*model.Method{{Name: "m", Args: []*model.Variable{nil}}},
		}},
	}

	if _, err := data.MarshalValue(project); err != nil {
		t.Fatalf("MarshalValue: %v", err)
	}
}
