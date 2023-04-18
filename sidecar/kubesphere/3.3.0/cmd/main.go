/*
Copyright 2020 The KubeSphere Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package main

import (
	"flag"
	"kubesphere/pkg/controller"
	"kubesphere/pkg/ks"
	"kubesphere/pkg/tenant"
	"log"
	"net/http"
	"os"
	"sync"
	"time"

	"k8s.io/klog"

	"github.com/emicklei/go-restful/v3"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"kubesphere.io/kubesphere/pkg/utils/signals"
)

const (
	ControllerTypeENV     = "Controller_Type"
	ControllerTypeWatch   = "watch"
	ControllerTypePolling = "polling"

	PollingIntervalENV     = "Polling_Interval"
	DefaultPollingInterval = time.Second * 5
)

var (
	kubeConfig       string
	stopCh           <-chan struct{}
	waitHandlerGroup sync.WaitGroup
)

func main() {

	cmd := NewServerCommand()

	if err := cmd.Execute(); err != nil {
		log.Fatalln(err)
	}
}

func AddFlags(fs *pflag.FlagSet) {
	fs.StringVar(&kubeConfig, "kubeconfig", "", "kubeconfig path")
}

func NewServerCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:  "kubesphere-tenant-sidecar",
		Long: `The sidecar to determining which tenant should receive notificaitons`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return Run()
		},
	}
	AddFlags(cmd.Flags())
	cmd.Flags().AddGoFlagSet(flag.CommandLine)

	return cmd
}

func Run() error {

	pflag.VisitAll(func(flag *pflag.Flag) {
		klog.Errorf("FLAG: --%s=%q", flag.Name, flag.Value)
	})

	stopCh = signals.SetupSignalHandler()

	r, err := ks.NewRuntime(kubeConfig, stopCh)
	if err != nil {
		klog.Errorf("start kubesphere runtime error, %s", err.Error())
		return err
	}

	controllerType := ControllerTypePolling
	if os.Getenv(ControllerTypeENV) == ControllerTypeWatch {
		controllerType = ControllerTypeWatch
	}

	if controllerType == ControllerTypeWatch {
		go controller.NewWatchController(r).Run(stopCh)
	} else {
		interval := DefaultPollingInterval
		i, err := time.ParseDuration(os.Getenv(PollingIntervalENV))
		if err == nil && i != 0 {
			interval = i
		}
		go controller.NewPollingController(r, interval).Run(stopCh)
	}

	return httpserver()
}

func httpserver() error {
	container := restful.NewContainer()
	ws := new(restful.WebService)
	ws.Path("").
		Consumes(restful.MIME_JSON).
		Produces(restful.MIME_JSON)
	ws.Route(ws.GET("/api/v2/tenant").To(handler))
	ws.Route(ws.GET("/readiness").To(readiness))
	ws.Route(ws.GET("/liveness").To(readiness))
	ws.Route(ws.GET("/preStop").To(preStop))

	container.Add(ws)

	server := &http.Server{
		Addr:    ":19094",
		Handler: container,
	}

	if err := server.ListenAndServe(); err != nil {
		klog.Fatal(err)
	}

	return nil
}

func handler(req *restful.Request, resp *restful.Response) {

	waitHandlerGroup.Add(1)
	defer waitHandlerGroup.Done()

	ns := req.QueryParameter("namespace")
	tenants := tenant.FromNamespace(ns)
	if tenants == nil {
		responseWithHeaderAndEntity(resp, http.StatusNotFound, "")
		return
	}

	responseWithJson(resp, tenants)
}

//readiness
func readiness(_ *restful.Request, resp *restful.Response) {

	responseWithHeaderAndEntity(resp, http.StatusOK, "")
}

//preStop
func preStop(_ *restful.Request, resp *restful.Response) {

	waitHandlerGroup.Wait()
	klog.Errorf("msg handler close, wait pool close")
	responseWithHeaderAndEntity(resp, http.StatusOK, "")
	klog.Flush()
}

func responseWithJson(resp *restful.Response, value interface{}) {
	e := resp.WriteAsJson(value)
	if e != nil {
		klog.Errorf("response error %s", e)
	}
}

func responseWithHeaderAndEntity(resp *restful.Response, status int, value interface{}) {
	e := resp.WriteHeaderAndEntity(status, value)
	if e != nil {
		klog.Errorf("response error %s", e)
	}
}
