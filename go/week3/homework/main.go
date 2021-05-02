package main

import (
	"context"
	"fmt"
	"golang.org/x/sync/errgroup"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
)

func startApp(ctx context.Context, addr string, handler http.Handler) error {
	svr := http.Server{
		Addr:    addr,
		Handler: handler,
	}

	go func() {
		<-ctx.Done()
		log.Printf("http server %v shutdown", addr)
		svr.Shutdown(context.Background())
	}()

	return svr.ListenAndServe()
}

func startSignal(ctx context.Context) error {
	sig := make(chan os.Signal)
	defer close(sig)
	signal.Notify(sig)
	for {
		select {
		case <-ctx.Done():
			log.Println("exit signal")
			return nil
		case s := <-sig:
			switch s {
			case syscall.SIGTERM, syscall.SIGHUP, syscall.SIGQUIT, syscall.SIGINT:
				log.Printf("receive signal: %v, and exit process", s)
				return fmt.Errorf("exit")
			default:
				log.Printf("receive signal: %v, continue", s)
			}
		}
	}
	return nil
}

func main() {
	log.Println("基于 errgroup 实现一个 http server 的启动和关闭 ，以及 linux signal 信号的注册和处理，要保证能够一个退出，全部注销退出。")

	cancelCtx, handleCancel := context.WithCancel(context.Background())

	g, ctx := errgroup.WithContext(cancelCtx)

	httpHandler := http.NewServeMux()
	httpHandler.HandleFunc("/shotdown", func(writer http.ResponseWriter, request *http.Request) {
		handleCancel()
		fmt.Fprintf(writer, "shotdwon")
	})

	g.Go(func() error {
		// http服务的生命周期是开源被管控的，通过ctx cancel同步http服务的shutdown
		return startApp(ctx, ":8000", httpHandler)
	})

	g.Go(func() error {
		return startSignal(ctx)
	})

	if err := g.Wait(); err != nil {
		log.Println(err)
	}
}
