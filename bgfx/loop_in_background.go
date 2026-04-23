package bgfx

import (
	"context"

	"github.com/Deimvis-go/vkbg/vkbg"
	"github.com/Deimvis/go-ext/go1.25/xoptional"
	"go.uber.org/fx"
)

// TODO: reimplement as RunAllTasksWith(s) func(lc fx.Lifecycle, ...)
// check if it works with generic fx parameters (LoopAllTasks[MyContext])

type RunSchedulerParams struct {
	fx.In

	S             *vkbg.SimpleScheduler
	Tasks         []vkbg.Task              `group:"bg_tasks"`
	OptionalTasks []xoptional.T[vkbg.Task] `group:"bg_tasks"`
}

// NOTE: very experimental thing
func RunScheduler(
	lc fx.Lifecycle,
	p RunSchedulerParams,
) {
	for _, t := range p.Tasks {
		p.S.MustSchedule(t)
	}
	for _, t := range p.OptionalTasks {
		if t.HasValue() {
			p.S.MustSchedule(t.Value())
		}
	}
	var cancel context.CancelFunc
	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			cancel = p.S.LoopInBackground()
			return nil
		},
		OnStop: func(ctx context.Context) error {
			cancel()
			return nil
		},
	})
}
