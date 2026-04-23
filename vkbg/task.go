package vkbg

type TaskAction func(*Context) error
type TaskMiddlewareFn func(*Context, TaskAction) error

type Task interface {
	Id() string
	Middlewares() []TaskMiddlewareFn
	Run(c *Context) error
}

func NewTask(id string, action TaskAction, mws ...TaskMiddlewareFn) Task {
	t := &TaskImpl{
		id:     id,
		action: action,

		middlewares: mws,
	}
	return t
}

type TaskImpl struct {
	id     string
	action TaskAction

	middlewares []TaskMiddlewareFn
}

func (t *TaskImpl) Id() string {
	return t.id
}

func (t *TaskImpl) Middlewares() []TaskMiddlewareFn {
	return t.middlewares
}

func (t *TaskImpl) Run(c *Context) error {
	return t.action(c)
}
