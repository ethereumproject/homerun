
The idea is to provide a program which will parse a directory
containing geth executables and configurations and automatically connect them
in a private network.

Chain Identity should be decided by convention by putting the executables and
configs in named subdirectories.

It should set each node to run on a different RPC port, and it should print
the ports each executable is exposing for RPC.

It should not enable mining on any of them by default; that can be switched on
with rpc or `geth attach`.

Base dir should look like:

```bash
basedir/ # homerun --dir=basedir

basedir/red/
basedir/red/geth-master # <- executable
basedir/red/chain.json # <- config
basedir/red/flags.conf # <- optional flags for this node

basedir/blue/
basedir/blue/geth-v3.5.86
basedir/blue/chain.json
basedir/blue/flags.conf

```

Can configure as many chains/instances like this as desired.
