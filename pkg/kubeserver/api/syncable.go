package api

type SyncableK8sBaseResourceListInput struct {
	SyncStatus []string `json:"sync_status"`
}
