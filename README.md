## Kube Drainer
Simple script to drain all kubernetes nodes with batches, and leave cluster autoscaler to remove unused nodes. Useful for replacing all EKS nodes with respecting PDB

#### Usage 
```shell
go run main.go -batch-size 2 -grace-period-seconds 120
```

#### TODO
- Interact with cloud provider to delete drained nodes
- pause if there is critical pods are not ready