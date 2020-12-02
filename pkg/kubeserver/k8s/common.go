package k8s

import (
	"encoding/json"
	"net/http"

	"github.com/ghodss/yaml"
	"k8s.io/apimachinery/pkg/runtime"

	"yunion.io/x/log"
)

func SendJSON(w http.ResponseWriter, obj interface{}) {
	w.Header().Set("Content-Type", "application/json")
	err := json.NewEncoder(w).Encode(obj)
	if err != nil {
		log.Errorf("Send obj %#v to http response error: %v", obj, err)
	}
}

func SendYAML(w http.ResponseWriter, obj runtime.Object) {
	w.Header().Set("Content-Type", "application/yaml")
	yaml.Marshal(obj)
	bytes, err := yaml.Marshal(obj)
	if err != nil {
		log.Errorf("Send obj %#v to http response error: %v", obj, err)
	}
	w.Write(bytes)
}
