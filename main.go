package main

import (
	"context"
	"flag"
	"fmt"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/kubectl/pkg/drain"
	"os"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
	"sync"
	"sync/atomic"
)

var batchSize int
var gracePeriodSeconds int

func main() {

	flag.IntVar(&batchSize, "batch-size", 2, "max number of nodes to drained in the same time")
	flag.IntVar(&gracePeriodSeconds, "grace-period-seconds", 60, "max seconds for pods eviction")
	flag.Parse()
	fmt.Println(batchSize, gracePeriodSeconds)
	// Get kubernetes client
	client, err := kubernetes.NewForConfig(config.GetConfigOrDie())
	if err != nil {
		panic(err)
	}

	// List all nodes
	nodes, err := client.CoreV1().Nodes().List(context.Background(), metav1.ListOptions{})
	if err != nil {
		panic(err)
	}

	// Get drain helper
	drainHelper := newDrainHelper(client)
	// Drain all nodes
	var wg WaitGroupCount
	for _, node := range nodes.Items {
		wg.Add(1)
		lnode := node
		go func() {
			fmt.Printf("draining node %s\n", lnode.Name)
			err = drainNode(drainHelper, &lnode, &wg)
			if err != nil {
				panic(err)
			}
		}()
		if wg.GetCount() >= batchSize {
			wg.Wait()
		}
	}
	wg.Wait()
}

func newDrainHelper(client kubernetes.Interface) *drain.Helper {
	drainHelper := &drain.Helper{}
	drainHelper.Ctx = context.Background()
	drainHelper.Client = client
	drainHelper.IgnoreAllDaemonSets = true
	drainHelper.DeleteEmptyDirData = true
	drainHelper.ErrOut = os.Stdout
	drainHelper.Out = os.Stdout
	drainHelper.GracePeriodSeconds = gracePeriodSeconds
	return drainHelper
}

func drainNode(drainHelper *drain.Helper, node *v1.Node, wg *WaitGroupCount) error {
	err := drain.RunCordonOrUncordon(drainHelper, node, true)
	if err != nil {
		return err
	}

	err = drain.RunNodeDrain(drainHelper, node.Name)
	if err != nil {
		return err
	}
	wg.Done()
	return nil
}

type WaitGroupCount struct {
	sync.WaitGroup
	count int64
}

func (wg *WaitGroupCount) Add(delta int) {
	atomic.AddInt64(&wg.count, int64(delta))
	wg.WaitGroup.Add(delta)
}

func (wg *WaitGroupCount) Done() {
	atomic.AddInt64(&wg.count, -1)
	wg.WaitGroup.Done()
}

func (wg *WaitGroupCount) GetCount() int {
	return int(atomic.LoadInt64(&wg.count))
}
