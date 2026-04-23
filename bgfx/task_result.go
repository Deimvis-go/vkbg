package bgfx

import (
	"github.com/Deimvis-go/vkbg/vkbg"
	"github.com/Deimvis/go-ext/go1.25/xoptional"
	"go.uber.org/fx"
)

type TaskResult struct {
	fx.Out

	Task vkbg.Task `group:"bg_tasks"`
}

type OptionalTaskResult struct {
	fx.Out

	Task xoptional.T[vkbg.Task] `group:"bg_tasks"`
}

var (
	NoTaskResult = OptionalTaskResult{}
)
