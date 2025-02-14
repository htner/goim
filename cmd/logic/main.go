package main

import (
	"context"
	"flag"
	"net"
	"os"
	"os/signal"
	"strconv"
	"syscall"

	"github.com/Terry-Mao/goim/internal/logic"
	"github.com/Terry-Mao/goim/internal/logic/conf"
	"github.com/Terry-Mao/goim/internal/logic/grpc"
	"github.com/Terry-Mao/goim/internal/logic/http"
	"github.com/Terry-Mao/goim/internal/logic/model"
	"github.com/Terry-Mao/goim/pkg/ip"
	consul "github.com/go-kratos/consul/registry"
	"github.com/go-kratos/kratos/v2/registry"
	"github.com/go-kratos/kratos/v2/transport/grpc/resolver/discovery"
	log "github.com/golang/glog"
	"github.com/google/uuid"
	"github.com/hashicorp/consul/api"
	"google.golang.org/grpc/resolver"
)

const (
	ver   = "2.0.0"
	appid = "goim.logic"
)

func main() {
	flag.Parse()
	if err := conf.Init(); err != nil {
		panic(err)
	}
	log.Infof("goim-logic [version: %s env: %+v] start", ver, conf.Conf.Env)
	// new consul client
	client, err := api.NewClient(api.DefaultConfig())
	if err != nil {
		panic(err)
	}
	// new reg with consul client
	reg := consul.New(client)
	resolver.Register(discovery.NewBuilder(reg))

	// logic
	srv := logic.New(conf.Conf)
	httpSrv := http.New(conf.Conf.HTTPServer, srv)
	rpcSrv := grpc.New(conf.Conf.RPCServer, srv)
	cancel := register(reg, srv)
	// signal
	c := make(chan os.Signal, 1)
	signal.Notify(c, syscall.SIGHUP, syscall.SIGQUIT, syscall.SIGTERM, syscall.SIGINT)
	for {
		s := <-c
		log.Infof("goim-logic get a signal %s", s.String())
		switch s {
		case syscall.SIGQUIT, syscall.SIGTERM, syscall.SIGINT:
			if cancel != nil {
				cancel()
			}
			srv.Close()
			httpSrv.Close()
			rpcSrv.GracefulStop()
			log.Infof("goim-logic [version: %s] exit", ver)
			log.Flush()
			return
		case syscall.SIGHUP:
		default:
			return
		}
	}
}

func register(dis registry.Registrar, srv *logic.Logic) context.CancelFunc {
	env := conf.Conf.Env
	addr := ip.InternalIP()
	_, port, _ := net.SplitHostPort(conf.Conf.RPCServer.Addr)
	/*
		ins := &naming.Instance{
			Region:   env.Region,
			Zone:     env.Zone,
			Env:      env.DeployEnv,
			Hostname: env.Host,
			AppID:    appid,
			Addrs: []string{
				"grpc://" + addr + ":" + port,
			},
			Metadata: map[string]string{
				model.MetaWeight: strconv.FormatInt(env.Weight, 10),
			},
		}
	*/

	ins := &registry.ServiceInstance{
		ID:      uuid.New().String(),
		Name:    appid,
		Version: ver,
		Metadata: map[string]string{
			model.MetaWeight: strconv.FormatInt(env.Weight, 10),
		},
		Endpoints: []string{
			"grpc://" + addr + ":" + port,
		},
	}

	err := dis.Register(context.Background(), ins)
	if err != nil {
		panic(err)
	}
	cancel := func() {
		dis.Deregister(context.Background(), ins)
	}
	return cancel
}
