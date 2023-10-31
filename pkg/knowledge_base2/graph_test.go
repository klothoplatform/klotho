package knowledgebase2

// import (
// 	"testing"

// 	construct "github.com/klothoplatform/klotho/pkg/construct2"
// )

// func Test_IsOperationalResourceSideEffect(t *testing.T) {
// 	type args struct {
// 		dag       construct.Graph
// 		templates []*ResourceTemplate
// 		rid       construct.ResourceId
// 		id        construct.ResourceId
// 	}
// 	tests := []struct {
// 		name    string
// 		args    args
// 		want    bool
// 		wantErr bool
// 	}{
// 		{},
// 	}
// 	for _, tt := range tests {
// 		t.Run(tt.name, func(t *testing.T) {
// 			kb := &KnowledgeBase{}
// 			for _, template := range tt.args.templates {
// 				kb.AddResourceTemplate(template)
// 			}

// 			got, err := IsOperationalResourceSideEffect(tt.args.dag, tt.args.kb, tt.args.rid, tt.args.id)
// 			if (err != nil) != tt.wantErr {
// 				t.Errorf("IsOperationalResourceSideEffect() error = %v, wantErr %v", err, tt.wantErr)
// 				return
// 			}
// 			if got != tt.want {
// 				t.Errorf(
// 					"IsOperationalResourceSideEffect() got = %v, want %v",
// 					got, tt.want,
// 				)
// 			}
// 		})
// 	}
// }
