package main

import (
	"bytes"
	"errors"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/ethereumproject/go-ethereum/rpc"
	"github.com/phayes/permbits"
	// "github.com/BurntSushi/toml"
)

var defaultRPCAPIMethods = []string{"admin", "eth", "net", "web3", "miner", "personal", "debug"}
var defaultCacheSize = 128
var defaultRPCPort = 8545
var defaultListenPort = 30303

var errConvertJSON = errors.New("Could not convert JSON response to golang data type")
var errRPCResponse = errors.New("No response from RPC")

var hrBaseDir string
var hrRPCDomain = "http://localhost"

type gethExec struct {
	Executable    string
	ChainIdentity string // set by containing subdir name
	Enode         string // set automatically
	// RPCPort       int    // set in homerun, with 8545 as reference default
	// ListenPort    int
	Client    rpc.Client
	ConfFlags []string // set with file anything.conf in chain subdir. should be just like a bash script but without the executable name. will parse just strings separated by spaces
}

func (g *gethExec) setEnode(s string) {
	g.Enode = s
}

func init() {
	flag.StringVar(&hrBaseDir, "dir", "", "base directory containing chain dirs")
	flag.StringVar(&hrRPCDomain, "rpc-domain", "http://localhost", "domain for geth rpc's")
}

func main() {
	flag.Parse()

	hrBaseDir = mustMakeDirPath(hrBaseDir)
	runs, err := collectChains(hrBaseDir)
	if err != nil {
		log.Fatalln(err)
	}

	log.Printf("Found %d chains...\n", len(runs))
	var dones = make(chan error)

	startNodes(runs, dones)
	connectNodes(runs)

	for _, r := range runs {
		log.Printf("Chain '%s' configured: %v", r.ChainIdentity, r.ConfFlags)
	}

	// block until dones closes (interrupt or error)
	<-dones

}

func startNodes(runs []*gethExec, dones chan error) {

	cmds := []*exec.Cmd{}

	go func() {
		select {
		case err := <-dones:
			if err != nil {
				log.Fatalln(err)
			}
		}
	}()

	for _, run := range runs {
		go func(run *gethExec) {
			log.Printf("Starting chain '%s'...\n", run.ChainIdentity)

			cmd := exec.Command(run.Executable, run.ConfFlags...)

			cmds = append(cmds, cmd)

			// capture helpful debugging error output
			var stderr bytes.Buffer
			cmd.Stderr = &stderr
			if e := cmd.Run(); e != nil {
				log.Printf("Chain '%s' error: %s: %s\n", run.ChainIdentity, e, stderr.String())
				dones <- e
			}
		}(run)
	}

	go func() {
		// sigc is a single-val channel for listening to program interrupt
		var sigc = make(chan os.Signal, 1)
		signal.Notify(sigc, os.Interrupt, syscall.SIGTERM)
		defer signal.Stop(sigc)
		sig := <-sigc
		log.Printf("Got %v, shutting down...", sig)
		for i, c := range cmds {
			if err := c.Process.Kill(); err != nil {
				log.Fatalln("Failed to kill", err)
			} else {
				log.Printf("Killed process %d\n", i)
			}
		}
		close(dones)
	}()

	// Wait for rpc to get up and running
	var ticker = time.Tick(time.Second)
	var done = make(chan (bool))
	haveEnodes := map[string]bool{}
	go func() {
		for {
			// since each entry is unique
			if len(haveEnodes) >= len(runs) {
				done <- true
			}
			select {
			case <-ticker:
				for _, run := range runs {
					if run.Enode != "" {
						haveEnodes[run.ChainIdentity] = true
						continue
					}
					resMap, err := run.rpcMap("admin_nodeInfo", []interface{}{})
					if err != nil {
						log.Println("no enode:", err)
						continue
					}
					run.setEnode(resMap["enode"].(string))
					log.Printf("Chain '%s': %s\n", run.ChainIdentity, run.Enode)
				}
			case <-done:
				break
			}
		}
	}()
	<-done
}

func connectNodes(runs []*gethExec) {
	log.Println("Connecting nodes...")
	for i, run := range runs {
		for j, run2 := range runs {
			if i < j && i != j {
				res, err := run.rpcBool("admin_addPeer", []string{run2.Enode})
				log.Println("Add peer", run.ChainIdentity, run2.ChainIdentity, res, err)
			}
		}
	}
}

func collectChains(basePath string) ([]*gethExec, error) {
	var runnables = []*gethExec{}

	chainDirs, err := ioutil.ReadDir(hrBaseDir)
	if err != nil {
		return runnables, err
	}

	for i, chain := range chainDirs {
		if !chain.IsDir() {
			log.Printf("Found non-directory: '%s', skipping...\n", chain.Name())
		}
		// log.Println("chain.Name()", chain.Name()) // eg. 'blue'

		executable := &gethExec{
			ChainIdentity: chain.Name(),
			// Client:        client, // established after configuration is parsed or assigned by default
		}

		files, err := ioutil.ReadDir(filepath.Join(hrBaseDir, chain.Name()))
		if err != nil {
			return runnables, err
		}
		for _, file := range files {
			if file.IsDir() {
				continue
			}
			fullFilename := filepath.Join(hrBaseDir, chain.Name(), file.Name())
			perms, e := permbits.Stat(fullFilename)
			if e != nil {
				return runnables, e
			}
			if perms.UserExecute() {
				if executable.Executable == "" {
					executable.Executable = fullFilename
				}
			}

			// set up custom flags from .conf file
			if filepath.Ext(file.Name()) == ".conf" {
				sNN, e := wordsFromFile(filepath.Join(hrBaseDir, chain.Name(), file.Name()))
				if e != nil {
					return runnables, e
				}
				executable.ConfFlags = sNN
			}
		}

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

		hasRpc := sliceContainsStrings(executable.ConfFlags, []string{"rpc"})
		if !hasRpc {
			log.Println(executable.ConfFlags)
			return runnables, errors.New("Chain '" + executable.ChainIdentity + "': RPC is required to be enabled.")
		}

		// set default rpc port if not set explicitly
		rpcPort := valueInSliceFollowingKey(executable.ConfFlags, []string{"rpcport", "rpc-port"})
		if rpcPort == "" {
			rpcPort = strconv.Itoa(defaultRPCPort)
		}

		c := valueInSliceFollowingKey(executable.ConfFlags, []string{"chain"})
		if c != "" {
			executable.ChainIdentity = c
		}

		client, err := rpc.NewClient(fmt.Sprintf("%s:%s", hrRPCDomain, rpcPort))
		if err != nil {
			return runnables, err
		}
		executable.Client = client
		log.Println("Create runnable: ", executable)
		runnables = append(runnables, executable)
	}
	return runnables, nil
}

func (g *gethExec) rpcBool(method string, params interface{}) (bool, error) {
	req := map[string]interface{}{
		"id":      new(int64),
		"method":  method,
		"jsonrpc": "2.0",
		"params":  params,
	}

	if err := g.Client.Send(req); err != nil {
		return false, err
	}

	var res rpc.JSONSuccessResponse
	if err := g.Client.Recv(&res); err != nil {
		return false, err
	}

	if res.Result != nil {
		mr, ok := res.Result.(bool)
		if ok {
			return mr, nil
		}
		return false, errConvertJSON
	}
	return false, errRPCResponse
}

func (g *gethExec) rpcMap(method string, params interface{}) (map[string]interface{}, error) {
	req := map[string]interface{}{
		"id":      new(int64),
		"method":  method,
		"jsonrpc": "2.0",
		"params":  params,
	}

	if err := g.Client.Send(req); err != nil {
		return nil, err
	}

	var res rpc.JSONSuccessResponse
	if err := g.Client.Recv(&res); err != nil {
		return nil, err
	}

	if res.Result != nil {
		mr, ok := res.Result.(map[string]interface{})
		if ok {
			return mr, nil
		}
		return nil, errConvertJSON
	}
	return nil, errRPCResponse
}

func wordsFromFile(filename string) ([]string, error) {
	b, e := ioutil.ReadFile(filename)
	if e != nil {
		return nil, e
	}
	bs := string(b)

	// extract only words from file, separating on whitespace and newlines
	re := regexp.MustCompile(`[\s\n\\]`)
	nonEmptyRe := regexp.MustCompile(`[\S]`)
	sN := re.Split(bs, -1)
	sNN := []string{}
	for _, s := range sN {
		ss := strings.TrimSpace(s)
		if ss != "" && nonEmptyRe.MatchString(ss) {
			sNN = append(sNN, ss)
		}
	}

	// for _, s := range sNN {
	// 	log.Println(" - ", s)
	// }
	return sNN, nil
}

func sliceContainsStrings(ss []string, s []string) bool {
	for _, x := range ss {
		for _, y := range s {
			if x == y {
				return true
			}
		}
	}
	return false
}

func valueInSliceFollowingKey(confFlags []string, keys []string) string {
	for i, s := range confFlags {
		for _, k := range keys {
			if s == k || strings.TrimPrefix(s, "-") == k || strings.TrimPrefix(s, "--") == k {
				// avoid out of bounds
				if i != len(confFlags)-1 {
					return confFlags[i+1]
				}
			}
		}
	}
	return ""
}

func mustMakeDirPath(p string) string {
	var e error
	if p == "" {
		p, e = os.Getwd()
		if e != nil {
			panic(e)
		}
	}
	abs, e := filepath.Abs(p)
	if e != nil {
		panic(e)
	}
	di, de := os.Stat(abs)
	if de != nil {
		panic(de)
	}
	if !di.IsDir() {
		panic("path must be a dir")
	}
	return abs
}
