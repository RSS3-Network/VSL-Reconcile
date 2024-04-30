# VSL Reconcile

An external state management tool for OP sequencers

## Environment Variables

### SEQUENCERS_LIST

`SEQUENCERS_LIST` is the comma separated sequencer list for OP nodes, it might look like this:

```
http://sequencer-1:9545,http://sequencer-2:9545,http://sequencer-3:9545
```

We recommend using internal DNS to access those pods, which should have better performance.

Please specify this parameter.

### CHECK_INTERVAL

`CHECK_INTERVAL` is the interval setting for each heartbeat loop call. It has a default value of `60s` 
which represents heartbeat check is executed per second, if you don't need such high frequency,
feel free to adjust this (like to `2m`).

### MAX_BLOCK_TIME

`MAX_BLOCK_TIME` is the maximal tolerate block time of the target Lay 2 chain. If the sequencer doesn't
produce new blocks and finally the same block height has been kept too long to exceeds this value,
the program then will switch the sequencer to another one.

The default value of this is `5m`, feel free to adjust it. But since it's check is based on heartbeat loop,
any value smaller than `CHECK_INTERVAL` might not work properly.

## Kubernetes configuration

To add more auth plugins, add below imports to `config/kube.go` :

```go
// Load all auth plugins
_ "k8s.io/client-go/plugin/pkg/client/auth"

// Load specific auth plugins
_ "k8s.io/client-go/plugin/pkg/client/auth/azure"
_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
_ "k8s.io/client-go/plugin/pkg/client/auth/oidc"
```
