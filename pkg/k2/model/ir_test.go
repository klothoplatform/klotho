package model

import (
	"reflect"
	"testing"
)

func createTestApplicationEnvironment() *ApplicationEnvironment {
	urnContainer, _ := ParseURN("urn:accountid:project:dev::construct/klotho.aws.Container:my-container")
	urnS3, _ := ParseURN("urn:accountid:project:dev::construct/klotho.aws.Bucket:my-bucket")
	appUrn, _ := ParseURN("urn:k2:app:myapp")
	projectUrn, _ := ParseURN("urn:k2:project:myproject")

	return &ApplicationEnvironment{
		SchemaVersion: 1,
		Version:       1,
		ProjectURN:    *projectUrn,
		AppURN:        *appUrn,
		Environment:   "dev",
		Constructs: map[string]Construct{
			"my-container": {
				URN:     urnContainer,
				Version: 1,
				Inputs: map[string]Input{
					"image": {
						Value:     "nginx:latest",
						Encrypted: false,
						Status:    InputStatusResolved,
						DependsOn: "",
					},
					"port": {
						Value:     80,
						Encrypted: false,
						Status:    InputStatusResolved,
						DependsOn: "",
					},
				},
			},
			"my-bucket": {
				URN:     urnS3,
				Version: 1,
			},
		},
	}
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
			name: "ValidFile",
			args: args{
				filename: "../ir_samples/testenv.yaml",
			},
			want:    createTestApplicationEnvironment(),
			wantErr: false,
		},
		{
			name: "NonExistentFile",
			args: args{
				filename: "../ir_samples/nonexistent.yaml",
			},
			want:    &ApplicationEnvironment{},
			wantErr: true,
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

func TestParseIRFile(t *testing.T) {
	validContent := `
schemaVersion: 1
version: 1
project_urn: "urn:k2:project:myproject"
app_urn: "urn:k2:app:myapp"
environment: "dev"
constructs:
  my-container:
    urn: "urn:accountid:project:dev::construct/klotho.aws.Container:my-container"
    version: 1
    inputs:
      image:
        value: "nginx:latest"
        encrypted: false
        status: "resolved"
      port:
        value: 80
        encrypted: false
        status: "resolved"
  my-bucket:
    urn: "urn:accountid:project:dev::construct/klotho.aws.Bucket:my-bucket"
    version: 1
`

	invalidContent := `
schemaVersion: 1
version: one
project_urn: "urn:k2:project:myproject"
`

	tests := []struct {
		name    string
		content []byte
		want    *ApplicationEnvironment
		wantErr bool
	}{
		{
			name:    "ValidContent",
			content: []byte(validContent),
			want:    createTestApplicationEnvironment(),
			wantErr: false,
		},
		{
			name:    "InvalidContent",
			content: []byte(invalidContent),
			want:    &ApplicationEnvironment{},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseIRFile(tt.content)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseIRFile() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ParseIRFile() = %v, want %v", got, tt.want)
			}
		})
	}
}
