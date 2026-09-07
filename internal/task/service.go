package task

import "context"

// repository is declared here, at the consumer, and not next to the concrete
// type. That keeps the seam narrow and lets a test pass a fake.
type repository interface {
	List(ctx context.Context) ([]Task, error)
	Create(ctx context.Context, in Input) (Task, error)
	Update(ctx context.Context, id int64, in Input) (Task, error)
	Delete(ctx context.Context, id int64) error
}

// Service holds the rules. It validates before it touches storage, so an
// invalid task never reaches the database CHECK constraint.
type Service struct {
	repo repository
}

func NewService(repo repository) *Service { return &Service{repo: repo} }

func (s *Service) List(ctx context.Context) ([]Task, error) {
	return s.repo.List(ctx)
}

func (s *Service) Create(ctx context.Context, in Input) (Task, error) {
	in.Normalize()
	if err := in.Validate(); err != nil {
		return Task{}, err
	}
	return s.repo.Create(ctx, in)
}

func (s *Service) Update(ctx context.Context, id int64, in Input) (Task, error) {
	in.Normalize()
	if err := in.Validate(); err != nil {
		return Task{}, err
	}
	return s.repo.Update(ctx, id, in)
}

func (s *Service) Delete(ctx context.Context, id int64) error {
	return s.repo.Delete(ctx, id)
}
