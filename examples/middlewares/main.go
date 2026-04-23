package middlewares

import (
	"fmt"
	"time"

	"go.uber.org/zap"

	"github.com/Deimvis-go/vkbg/vkbg"
)

func MyTask(c *vkbg.Context) error {
	fmt.Println("start my task")
	time.Sleep(time.Second)
	fmt.Println("finish my task")
	return nil
}

func main() {
	logger := newLogger()
	s := vkbg.NewSimpleScheduler(
		vkbg.WithLogger(logger),
		vkbg.WithRecovery(logger),
	)
	// schedule task to run every minute
	s.MustSchedule(vkbg.NewTask("my_task", MyTask, vkbg.WithInterval(time.Minute)))
	cancel := s.LoopInBackground()
	defer cancel()
	time.Sleep(2 * time.Second)
}

func newLogger() *zap.SugaredLogger {
	config := zap.NewProductionConfig()
	config.EncoderConfig.TimeKey = ""
	logger, err := config.Build()
	if err != nil {
		panic(err)
	}
	defer logger.Sync()
	return logger.Sugar()
}
