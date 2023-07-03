package registry

import "testing"

func TestMirrorImage(t *testing.T) {
	type args struct {
		name   string
		tag    string
		prefix string
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "all",
			args: args{
				name:   "node",
				tag:    "v3.7.2",
				prefix: "calico",
			},
			want: "registry.cn-beijing.aliyuncs.com/yunionio/calico-node:v3.7.2",
		},
		{
			name: "no prefix",
			args: args{
				name:   "node",
				tag:    "v3.7.2",
				prefix: "",
			},
			want: "registry.cn-beijing.aliyuncs.com/yunionio/node:v3.7.2",
		},
		{
			name: "no tag",
			args: args{
				name:   "node",
				tag:    "",
				prefix: "",
			},
			want: "registry.cn-beijing.aliyuncs.com/yunionio/node:latest",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := MirrorImage("", tt.args.name, tt.args.tag, tt.args.prefix); got != tt.want {
				t.Errorf("MirrorImage() = %v, want %v", got, tt.want)
			}
		})
	}
}
