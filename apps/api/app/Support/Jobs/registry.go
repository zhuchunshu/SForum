package jobs

import "github.com/riverqueue/river"

type Registrar func(workers *river.Workers) error

type Registry struct {
	registrars []Registrar
}

func NewRegistry() *Registry {
	return &Registry{}
}

func (r *Registry) Add(registrar Registrar) {
	r.registrars = append(r.registrars, registrar)
}

func (r *Registry) Build() (*river.Workers, error) {
	workers := river.NewWorkers()
	for _, registrar := range r.registrars {
		if err := registrar(workers); err != nil {
			return nil, err
		}
	}
	return workers, nil
}
