package main

import (
	"flag"
	"os"
	"os/exec"
	"runtime/debug"
	"strconv"

	"go-gin-payment/cmd/cmd_lib"
	"go-gin-payment/config"
	"go-gin-payment/ext/logger"
	"go-gin-payment/jobs/api"

	_ "go-gin-payment/docs"
)

func main() {
	RunWithRecover(createProcess)
}

func createProcess() {
	e := flag.String("e", "development", "production | development")
	flag.Parse()

	cleaner := cmd_lib.SetupLog(*e)
	defer cleaner()

	if os.Getenv("_IS_CHILD") == "" {
		logger.L.Println("starting...")
		logger.L.Println("parent pid", os.Getpid())
		envs := os.Environ()
		envs = append(envs, "_IS_CHILD=1")
		cmd := exec.Command(os.Args[0], os.Args[1:]...)
		cmd.Env = envs
		err := cmd.Start()
		if err != nil {
			logger.L.Fatal(err)
		}
		p := cmd.Process.Pid
		logger.L.Infof("child started: %d, parent exit!", p)
		writePid(p)
	} else {
		logger.L.Println("env:", *e)
		logger.L.Println("is in docker:", os.Getenv("IS_IN_DOCKER"))

		// mysql / redis
		closer := cmd_lib.Prepare()
		defer closer()

		r := api.RunAPI()
		err := r.Run(config.APIPort)
		if err != nil {
			panic(err)
		}
		logger.L.Println("start Web API at:", config.APIPort)
	}
}

func writePid(pid int) {
	d := []byte(strconv.Itoa(pid))
	_ = os.WriteFile("run.pid", d, 0644)
}

// RunWithRecover函数用于在执行worker函数时，如果发生panic，则进行recover操作，并打印错误信息
func RunWithRecover(worker func()) {
	//defer关键字用于延迟执行后面的函数，这里用于在worker函数执行完毕后，进行recover操作
	defer func() {
		//recover函数用于捕获panic，如果发生panic，则返回panic的值，否则返回nil
		if err := recover(); err != nil {
			logger.L.Printf("FATAL in routine: %s\n%s", err, debug.Stack())
		}
	}()
	worker()
}
