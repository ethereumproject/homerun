
A small program to automate setting up a private network
for go-ethereum integration and development tests.

## Usage

```bash
# Create a base directory to hold all nodes and node data.
# This will behave like geth's 'data-dir'.
mkdir dev-mopo

# Create subdirectories for each desired node.
# The names of the directories will be the each node's _chain identity.
mkdir dev-mopo/red dev-mopo/blue

# Add the executables you want to run in the network
cp ~/bin/geth dev-mopo/red/geth
cp ~/bin/parity dev-mopo/blue/parity

# Create external chain configurations for each node.
# This is necessary for Geth Classic, but may differ across clients.
vim dev-mopo/red/chain.json dev-mopo/blue/chain.json

# Optionally: create external flag configuration files...
# This file is required to be suffixed with '.conf', but can be named
# anything else otherwise.
# If this file does not exist, the program will supply default
# flags (described below).
vim dev-mopo/red/--flags.conf dev-mopo-blue/in-here.conf

# All set!
homerun --dir dev-mopo
> 2017/08/23 11:46:35 Found 2 chains...
2017/08/23 11:46:35 Starting chain 'blue'...
2017/08/23 11:46:35 Starting chain 'red'...
2017/08/23 11:46:36 Chain 'blue': enode://5a4c7eec173de548bddd4c4c36d6d97e72227df7fd85604c7cf89b97fa00e386672243ee3fcae882f1a7fd1dbe4b4e8fbc3c58ef7992285924d7eacb7311489e@[::]:30305?discport=0
2017/08/23 11:46:36 Chain 'red': enode://f02e04da05f18a94dc0a70609e9871c4a4c43b2c9d1fbda37cde7b206c2f7406382fde4663f28e7aafa6f40e9c70785fd56cdab5e96e268f6036f899573d9b76@[::]:30304?discport=0
2017/08/23 11:46:37 Connecting nodes...
2017/08/23 11:46:37 Add peer blue red true <nil>
2017/08/23 11:46:37 Chain 'blue' configured: [--datadir /Users/ia/gocode/src/github.com/ethereumproject/homerun/testdir --chain blue --no-discover --port 30305 --rpc --rpcport 8555 --cache 129 --rpcapi admin,eth,net,web3]
2017/08/23 11:46:37 Chain 'red' configured: [--datadir /Users/ia/gocode/src/github.com/ethereumproject/homerun/testdir --chain red --nodiscover --port 30304 --rpc --rpcport 8546 --cache 128 --rpcapi admin,eth,net,web3,miner,personal,debug --log-dir /Users/ia/gocode/src/github.com/ethereumproject/homerun/testdir/red/logs]


# Interrupt the process with ^C
# Will also kill associated geth/client processes
^C2017/08/23 11:44:24 Got interrupt, shutting down...
2017/08/23 11:44:24 Killed process 0
2017/08/23 11:44:24 Killed process 1

```


## Node configurations

Each node is required to enable RPC. If custom flags are used without
enabling RPC, the program will exit or fail to establish an enode address
for the given client.

RPC ports and listen address ports will be automatically incremented from from
the default `:8545` if not explicitly configured in the flags.conf file.

The default flags are as follows:

```go
        // set default configuration if not configured by .conf file
        if executable.ConfFlags == nil {
            executable.ConfFlags = []string{
                "--datadir", hrBaseDir,
                "--chain", executable.ChainIdentity,
                "--nodiscover",
                "--port", strconv.Itoa(defaultListenPort + i),
                "--rpc",
                "--rpcport", strconv.Itoa(defaultRPCPort + i),
                "--cache", strconv.Itoa(defaultCacheSize),
                "--rpcapi", strings.Join(defaultRPCAPIMethods, ","),
                "--log-dir", filepath.Join(hrBaseDir, executable.ChainIdentity, "logs"),
            }
        }
```


## Behind the scenes

The program is very simple.

It's core behaviors are:

- run the executables with default or configured flags
- use RPC `admin_nodeInfo` to get the enode for each client
- use RPC `admin_addPeer` to connect each client

It does not enable mining on any of them by default; that can be switched on
with rpc or `geth attach`.

Overall, the base directory should look like:

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

