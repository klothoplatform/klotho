package model

import (
	"reflect"
	"testing"
)

// Create a simple ApplicationEnvironment for testing
func createTestApplicationEnvironment() *ApplicationEnvironment {
	urnContainer, _ := ParseURN("urn:accountid:project:dev::construct/klotho.aws.Container:my-container")
	urnS3, _ := ParseURN("urn:accountid:project:dev::construct/klotho.aws.Bucket:my-bucket")

	ae := &ApplicationEnvironment{
		SchemaVersion: 1,
		Version:       1,
		ProjectURN:    "urn:k2:project:myproject",
		AppURN:        "urn:k2:app:myapp",
		Environment:   "dev",
		Constructs: map[string]Construct{
			"my-container": {
				URN:     urnContainer,
				Version: 1,
				Inputs: map[string]Input{
					"image": {
						Value:     "nginx:latest",
						Encrypted: false,
						Status:    Resolved,
						DependsOn: nil,
					},
					"port": {
						Value:     80,
						Encrypted: false,
						Status:    Resolved,
						DependsOn: nil,
					},
				},
			},
			"my-bucket": {
				URN:     urnS3,
				Version: 1,
			},
		},
	}
	return ae
}

func TestReadIRFile(t *testing.T) {
	type args struct {
		filename string
	}
	tests := []struct {
		name    string
		args    args
		want    *ApplicationEnvironment
		wantErr bool
	}{
		{
			name: "TestReadIRFile",
			args: args{
				filename: "../ir_samples/testenv.yaml",
			},
			want:    createTestApplicationEnvironment(),
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ReadIRFile(tt.args.filename)
			if (err != nil) != tt.wantErr {
				t.Errorf("ReadIRFile() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ReadIRFile() = %v, want %v", got, tt.want)
			}
		})
	}
}
