package api

type PriorityClassCreateInput struct {
	ClusterResourceCreateInput

	Value         int32  `json:"value"`
	GlobalDefault bool   `json:"globalDefault"`
	Description   string `json:"description"`
}

type PriorityClassDetail struct {
	ClusterResourceDetail

	Value         int32  `json:"value"`
	GlobalDefault bool   `json:"globalDefault"`
	Description   string `json:"description"`
}
