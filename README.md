# VK Background

VK Background (vkbg) is a Golang package that implements flexible background tasks configuration.

## Features

* Stateful Context
* Middlewares
* Comprehensive task framework with 4 generic task events: started, failed, aborted, completed
* Off-the-shelf solutions, like PostgreSQL logging middleware

## Quick Start

```golang
package quick_start

import (
    "fmt"
    "time"

    "github.com/Deimvis-go/vkbg/vkbg"
)

func MyTask(c *vkbg.Context) error {
    fmt.Println("start my task")
    time.Sleep(time.Second)
    fmt.Println("finish my task")
    return nil
}

func main() {
    s := vkbg.NewSimpleScheduler()
    // schedule task to run every minute
    s.MustSchedule(vkbg.NewSimpleTask("my_task", MyTask, time.Minute))
    // loop in background
    cancel := s.LoopInBackground()
    defer cancel()
    time.Sleep(2 * time.Second)
}
```

* More examples in [examples](./examples) folder

