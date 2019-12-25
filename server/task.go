package server

import (
	"errors"
	"github.com/donnie4w/go-logger/logger"
	"insane/general/base/appconfig"
	"strconv"
	"sync"
	"time"

	"insane/utils"
)

type TaskList struct {
	UnfinishedTasks sync.Map // 待执行任务列表
	RunTasks        sync.Map // 正在执行的任务
	CompletedTasks  sync.Map // 已完成任务列表
	CurTask         chan *Task
	M               sync.Mutex
}

type Task struct {
	Request  *Request
	initOnce sync.Once
	stop     chan int
}

var TK = &TaskList{
	CurTask: make(chan *Task),
}

const (
	COMPLETED_TASK  = 1
	UNFINISHED_TASK = 2
	RUN_TASK        = 3
)

func (taskList *TaskList) TaskListAdd(request *Request) (err error) {
	if err = request.VerifyParam(); err != nil {
		return
	}
	task := &Task{
		Request: request,
	}
	task.initOnce.Do(func() {
		task.Init()
	})
	taskList.setTasks(task.Request.Id, task, UNFINISHED_TASK)
	return
}

func (taskList *TaskList) TaskListRemove(id string) (err error) {
	if v, ok := taskList.getTasks(id, RUN_TASK); ok {
		if err := v.Stop(); err != nil {
			logger.Debug(err)
		}
		return
	}
	if _, ok := taskList.getTasks(id, UNFINISHED_TASK); ok {
		taskList.deleteTasks(id, UNFINISHED_TASK)
		return
	}
	if _, ok := taskList.getTasks(id, COMPLETED_TASK); ok {
		taskList.deleteTasks(id, COMPLETED_TASK)
		return
	}

	return errors.New("任务不存在")
}

// 指定时间后，删除完成的任务
func (taskList *TaskList) TaskListTickerRemove(id string) {
	go func() {
		t := time.After(time.Duration(appconfig.GetConfig().Worker.TaskLife) * time.Second)
		<-t
		taskList.TaskListRemove(id)
	}()
}

// 停止任务
//func (taskList *TaskList) TaskListStop(id string) (err error) {
//	if v, ok := taskList.getTasks(id, UNFINISHED_TASK); ok {
//		err = v.Stop()
//	} else {
//		err = errors.New("任务不存在")
//	}
//	return
//}

func (taskList *TaskList) TaskListInfo(id string) (content string) {
	if v, ok := taskList.getTasks(id, RUN_TASK); ok {
		content = v.Info()
		return
	}
	if v, ok := taskList.getTasks(id, UNFINISHED_TASK); ok {
		content = v.Info()
		return
	}
	if v, ok := taskList.getTasks(id, COMPLETED_TASK); ok {
		content = v.Info()
		return
	}
	return
}

func (taskList *TaskList) TaskListStatus(id string) (status uint32) {
	if _, ok := taskList.CompletedTasks.Load(id); ok {
		return COMPLETED_TASK
	}
	if _, ok := taskList.RunTasks.Load(id); ok {
		return RUN_TASK
	}
	if _, ok := taskList.UnfinishedTasks.Load(id); ok {
		return UNFINISHED_TASK
	}
	return
}

func (taskList *TaskList) setTasks(id string, task *Task, tp uint32) {
	switch tp {
	case COMPLETED_TASK:
		taskList.UnfinishedTasks.Delete(id)
		taskList.RunTasks.Delete(id)

		taskList.CompletedTasks.Store(id, task)
	case UNFINISHED_TASK:
		taskList.CompletedTasks.Delete(id)
		taskList.RunTasks.Delete(id)

		taskList.UnfinishedTasks.Store(id, task)
	case RUN_TASK:
		taskList.CompletedTasks.Delete(id)
		taskList.UnfinishedTasks.Delete(id)

		taskList.RunTasks.Store(id, task)
	}
}

func (taskList *TaskList) getTasks(id string, tp uint32) (task *Task, ok bool) {
	var temp interface{}
	switch tp {
	case COMPLETED_TASK:
		temp, ok = taskList.CompletedTasks.Load(id)
	case UNFINISHED_TASK:
		temp, ok = taskList.UnfinishedTasks.Load(id)
	case RUN_TASK:
		temp, ok = taskList.RunTasks.Load(id)
	}
	task, ok = temp.(*Task)
	return
}

func (taskList *TaskList) getTasksAll(tp uint32) (tasks map[string]*Task) {
	var data sync.Map
	tasks = make(map[string]*Task)
	switch tp {
	case COMPLETED_TASK:
		data = taskList.CompletedTasks
	case UNFINISHED_TASK:
		data = taskList.UnfinishedTasks
	}
	data.Range(func(key, value interface{}) bool {
		k, ok1 := key.(string)
		v, ok2 := value.(*Task)
		if ok1 && ok2 {
			tasks[k] = v
		}
		return ok1 && ok2
	})
	return
}

func (taskList *TaskList) deleteTasks(id string, tp uint32) {
	switch tp {
	case COMPLETED_TASK:
		taskList.CompletedTasks.Delete(id)
	case UNFINISHED_TASK:
		taskList.UnfinishedTasks.Delete(id)
	case RUN_TASK:
		taskList.RunTasks.Delete(id)
	}
}

func (taskList *TaskList) TaskListRun() {
	go func() {
		for {

			// 没任务休息一会，避免占用CPU
			if len(taskList.getTasksAll(UNFINISHED_TASK)) < 1 {
				time.Sleep(30 * time.Millisecond)
			}

			// 循环任务列表，插入最新的任务ID
			// 每次只能插入一条任务且任务被消费才能继续插入任务
			for _, v := range taskList.getTasksAll(UNFINISHED_TASK) {
				taskList.setTasks(v.Request.Id, v, RUN_TASK) // 任务加入到正在运行任务
				taskList.CurTask <- v
			}
		}
	}()

	for {
		task := <-taskList.CurTask // 准备执行的任务id
		task.Run()
		taskList.setTasks(task.Request.Id, task, COMPLETED_TASK) // 任务加入到已完成任务列表
		taskList.TaskListTickerRemove(task.Request.Id)           // 已完成任务列表定时删除
	}
}

func (task *Task) Init() {
	id := strconv.FormatInt(utils.Now(), 10)
	task.Request.Id = id
	task.Request.Report = new(Report)
	task.Request.initStopCh()
}

func (task *Task) Run() {
	task.initOnce.Do(func() {
		task.Init()
	})
	task.Request.Dispose()
}

func (task *Task) Stop() error {
	if task.Request.Status {
		return nil
	}
	task.Request.Status = true
	return task.Request.Close()
}

func (task *Task) Info() string {
	return task.Request.Report.Get()
}
