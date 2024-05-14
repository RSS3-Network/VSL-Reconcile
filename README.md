# VSL Reconcile

VSL Reconcile serves as an external state management tool for OP sequencers.
It automatically transitions from an unhealthy sequencer to a backup sequencer to minimize downtime.

## Environment Variables

### SEQUENCERS_LIST

`SEQUENCERS_LIST` is the comma-separated sequencer list for OP nodes:

```
http://sequencer-1:9545,http://sequencer-2:9545,http://sequencer-3:9545
```

We recommend using internal DNS for discovering sequencers.

### CHECK_INTERVAL

`CHECK_INTERVAL` is the interval between heartbeat checks. Default: `60s`.

### MAX_BLOCK_TIME

`MAX_BLOCK_TIME` is the maximum amount of block time tolerated before a sequencer is deemed unhealthy. Default: `5m`. Must be longer than `CHECK_INTERVAL`.
Should the sequencer be unable to produce a block for a duration exceeding `MAX_BLOCK_TIME`, Reconcile will automatically switch to a backup sequencer listed in the `SEQUENCERS_LIST`.
