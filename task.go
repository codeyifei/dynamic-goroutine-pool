package dynamic_goroutine_pool

type TaskHandler func(v ...interface{}) error
type HandlerFunc func() error

type Task struct {
	Handler TaskHandler
	Params  []interface{}
}

func (t *Task) ParseHandler() HandlerFunc {
	return func() error {
		return t.Handler(t.Params...)
	}
}
