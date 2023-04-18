package controller

import (
	"k8s.io/klog"
	"kubesphere/pkg"
	"kubesphere/pkg/ks"
	"kubesphere/pkg/tenant"
	"time"
)

type PollingController struct {
	*ks.Runtime
	interval time.Duration
}

func NewPollingController(r *ks.Runtime, interval time.Duration) pkg.Controller {
	return &PollingController{
		Runtime:  r,
		interval: interval,
	}
}

func (c PollingController) Run(stopCh <-chan struct{}) {
	ticker := time.NewTicker(c.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			if err := tenant.Reload(c.Runtime); err != nil {
				klog.Errorf("reload tenant error, %s", err)
			}
		case <-stopCh:
			return
		}
	}
}
