package main

import (
	"bytes"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"os"
)

// const (
//     dataDir := "/tmp"
// )
//
// // For now, just use master geth classic.
// // TODO: incorporate other clients/+versions
// var (
//     gethStdCommands = map[string]string{
//         "data-dir": dataDir,
//         "rpc": "",
//         "rpc-api": "eth,admin,debug,miner,net,web3,txpool",
//         "cache": "128",
//         "no-discover": "",
//     }
// )
type rpcReq struct {
	JSONRPC string   `json:"jsonrpc"`
	Method  string   `json:"method"`
	Params  []string `json:"params"`
	ID      string   `json:"id"`
}

// type rpcRes struct {
// 	JSONRPC string                 `json:"jsonrpc"`
// 	ID      string                 `json:"id"`
// 	Result  map[string]interface{} `json:"result"`
// }

type rpcRes struct {
	Result map[string]interface{} `json:"result"`
}

func main() {
	r := rpcReq{
		JSONRPC: "2.0",
		Method:  "admin_nodeInfo",
		ID:      "1",
	}
	b := new(bytes.Buffer)
	json.NewEncoder(b).Encode(r)
	res, err := http.Post("http://localhost:8545", "application/json; charset=utf-8", b)
	if err != nil {
		log.Println(err)
		return
	}
	io.Copy(os.Stdout, res.Body)

	decRes := rpcRes{}
	json.NewDecoder(res.Body).Decode(decRes)
	log.Println(decRes)

	// body, readerr := ioutil.ReadAll(res.Body)
	// if readerr != nil {
	// 	log.Fatal(readerr)
	// }
	//
	// jsonerr := json.Unmarshal(body, &decRes)
	// if jsonerr != nil {
	// 	log.Fatal(jsonerr)
	// }
	//
	// log.Println(decRes)
}
