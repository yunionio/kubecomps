package client

import (
	"fmt"
	"net/http"
	"testing"
)

func Test_parseCatalogLink(t *testing.T) {
	tests := []struct {
		header http.Header
		want   string
	}{
		{
			header: http.Header{
				"Link": []string{"</v2/_catalog?last=yunionio%2Fcloudpods-ee&n=100>; rel=\"next\""},
			},
			want: "/v2/_catalog?last=yunionio%2Fcloudpods-ee&n=100",
		},
		{
			header: http.Header{},
			want:   "",
		},
	}
	for _, tt := range tests {
		t.Run(fmt.Sprintf("%#v", tt.header), func(t *testing.T) {
			if got := parseCatalogLink(tt.header); got != tt.want {
				t.Errorf("parseCatalogLink() = %v, want %v", got, tt.want)
			}
		})
	}
}
