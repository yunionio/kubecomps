package kubespray

type CreateRunner interface {
	Run() error
	// GetOutput get ansible playbook output
	GetOutput() string
}
