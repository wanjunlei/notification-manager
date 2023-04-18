package pkg

type Controller interface {
	Run(stopCh <-chan struct{})
}
