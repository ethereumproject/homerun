
The idea is to provide a program which will parse a directory
containing geth executables and configurations and automatically connect them
in a private network.

Chain Identity should be decided by convention by putting the executables and
configs in named subdirectories.

It should set `--max-peers` to the number of total nodes that will run.

It should set each node to run on a different RPC port, and it should print
the ports each executable is exposing for RPC.

It should not enable mining on any of them by default.


Base dir should look like:

```bash
basedir/ # homerun --dir=basedir

basedir/red/
basedir/red/geth-master # <- executable
basedir/red/chain.json # <- config
basedir/red/flags.toml # <- optional flags for this node

basedir/blue/
basedir/blue/geth-v3.5.86
basedir/blue/chain.json
basedir/blue/flags.toml

# Planned...
# basedir/yellow/
# basedir/yellow/geth-hf
# basedir/yellow/flags.toml

```
