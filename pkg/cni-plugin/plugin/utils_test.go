package plugin

import (
	"reflect"
	"testing"
)

func Test_parseCNIArgs(t *testing.T) {
	tests := []struct {
		args    string
		want    *PodInfo
		wantErr bool
	}{
		{
			args: "IgnoreUnknown=1;K8S_POD_NAMESPACE=27c9464ab54947328a29298761895be3;K8S_POD_NAME=test-pod5;K8S_POD_INFRA_CONTAINER_ID=c73d1df43df96b6804330a257855a5c2c8355d3f84019c57bcc8b5ede14a11ed;K8S_POD_UID=e25e38ef-fe98-4993-8641-699cd0530fc0",
			want: &PodInfo{
				Namespace:   "27c9464ab54947328a29298761895be3",
				Name:        "test-pod5",
				ContainerId: "c73d1df43df96b6804330a257855a5c2c8355d3f84019c57bcc8b5ede14a11ed",
				Id:          "e25e38ef-fe98-4993-8641-699cd0530fc0",
			},
			wantErr: false,
		},
		{
			args:    "",
			want:    nil,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.args, func(t *testing.T) {
			got, err := NewPodInfoFromCNIArgs(tt.args)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewPodInfoFromCNIArgs() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("NewPodInfoFromCNIArgs() got = %v, want %v", got, tt.want)
			}
		})
	}
}
