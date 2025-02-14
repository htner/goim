package main

import (
	"flag"
	"os"
	"os/signal"
	"syscall"

	"github.com/Terry-Mao/goim/internal/job/conf"

	"google.golang.org/grpc/resolver"

	consul "github.com/go-kratos/consul/registry"
	"github.com/go-kratos/kratos/v2/transport/grpc/resolver/discovery"

	"github.com/Terry-Mao/goim/internal/job"
	log "github.com/golang/glog"
	"github.com/hashicorp/consul/api"
)

var (
	ver = "2.0.0"
)

func main() {
	flag.Parse()
	if err := conf.Init(); err != nil {
		panic(err)
	}
	log.Infof("goim-job [version: %s env: %+v] start", ver, conf.Conf.Env)
	// new consul client
	client, err := api.NewClient(api.DefaultConfig())
	if err != nil {
		panic(err)
	}
	// new reg with consul client
	reg := consul.New(client)
	resolver.Register(discovery.NewBuilder(reg))
	// job
	j := job.New(conf.Conf)
	go j.Consume()
	// signal
	c := make(chan os.Signal, 1)
	signal.Notify(c, syscall.SIGHUP, syscall.SIGQUIT, syscall.SIGTERM, syscall.SIGINT)
	for {
		s := <-c
		log.Infof("goim-job get a signal %s", s.String())
		switch s {
		case syscall.SIGQUIT, syscall.SIGTERM, syscall.SIGINT:
			j.Close()
			log.Infof("goim-job [version: %s] exit", ver)
			log.Flush()
			return
		case syscall.SIGHUP:
		default:
			return
		}
	}
}
