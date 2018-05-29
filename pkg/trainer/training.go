package trainer

// TrainingJobInterface is an interface for training job management.
type TrainingJobInterface interface {
}

type trainingJob struct {
}

// NewJob creates a trainingJob which implements the TrainingJobInterface.
func NewJob() (TrainingJobInterface, error) {
	return &trainingJob{}, nil
}
