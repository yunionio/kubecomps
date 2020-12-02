package models

import (
	"reflect"
	"testing"
)

func TestMergeValues(t *testing.T) {
	type args struct {
		yamlStr string
		sets    map[string]string
	}
	tests := []struct {
		name    string
		args    args
		want    map[string]interface{}
		wantErr bool
	}{
		{
			name: "parse yaml string",
			args: args{
				yamlStr: `a: b
image:
  repo: aliyun`,
			},
			want: map[string]interface{}{
				"a": "b",
				"image": map[string]interface{}{
					"repo": "aliyun",
				},
			},
		},
		{
			name: "parse sets",
			args: args{
				sets: map[string]string{
					"image.pullPolicy": "IfNotPresent",
					"image.tag":        "v1",
				},
			},
			want: map[string]interface{}{
				"image": map[string]interface{}{
					"pullPolicy": "IfNotPresent",
					"tag":        "v1",
				},
			},
		},
		{
			name: "parse yaml str and sets",
			args: args{
				yamlStr: `a: b
image:
  repo: aliyun`,
				sets: map[string]string{
					"image.pullPolicy": "IfNotPresent",
					"image.tag":        "v1",
				},
			},
			want: map[string]interface{}{
				"a": "b",
				"image": map[string]interface{}{
					"pullPolicy": "IfNotPresent",
					"tag":        "v1",
					"repo":       "aliyun",
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := MergeValues(tt.args.yamlStr, tt.args.sets)
			if (err != nil) != tt.wantErr {
				t.Errorf("MergeValues() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("MergeValues() got = %v, want %v", got, tt.want)
			}
		})
	}
}
