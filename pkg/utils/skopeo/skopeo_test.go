package skopeo

import "testing"

func Test_skopeo_getCopyCommand(t *testing.T) {
	tests := []struct {
		name string
		args *CopyParams
		want string
	}{
		{
			name: "basic copy",
			args: &CopyParams{
				SrcTLSVerify: false,
				SrcUsername:  "docker",
				SrcPassword:  "docker",
				SrcPath:      "10.127.190.187:8083/lzx/influxdb:1.7.7-new",
				TargetPath:   "/tmp/influxdb.tar",
			},
			want: "skopeo copy --src-tls-verify=false --src-username docker --src-password docker docker://10.127.190.187:8083/lzx/influxdb:1.7.7-new docker-archive:/tmp/influxdb.tar",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &skopeo{}
			if got := s.getCopyCommand(tt.args); got != tt.want {
				t.Errorf("getCopyCommand() = %v, want %v", got, tt.want)
			}
		})
	}
}
