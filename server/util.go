package server

import (
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"github.com/gorilla/schema"
	"fmt"
	"strings"
)

var httpWriterSetContentType = func(w http.ResponseWriter, value string) {
	w.Header().Set("Content-Type", value)
}
var logPrintf = log.Printf
var lookupHost = net.LookupHost
var readFile = ioutil.ReadFile
var decoder = schema.NewDecoder()
var SendDistributeRequests = func(req *http.Request, port, proxyServiceName string) (status int, err error) {
	values := req.URL.Query()
	values.Set("distribute", "false")
	req.URL.RawQuery = values.Encode()
	dns := fmt.Sprintf("tasks.%s", proxyServiceName)
	failedDns := []string{}
	method := req.Method
	body := ""
	if req.Body != nil {
		defer func() { req.Body.Close() }()
		reqBody, _ := ioutil.ReadAll(req.Body)
		body = string(reqBody)
	}
	if ips, err := lookupHost(dns); err == nil {
		for i := 0; i < len(ips); i++ {
			req.URL.Host = fmt.Sprintf("%s:%s", ips[i], port)
			client := &http.Client{}
			addr := fmt.Sprintf("http://%s:%s%s?%s", ips[i], port, req.URL.Path, req.URL.RawQuery)
			logPrintf("Sending distribution request to %s", addr)
			req, _ := http.NewRequest(method, addr, strings.NewReader(body))
			if resp, err := client.Do(req); err != nil || resp.StatusCode >= 300 {
				failedDns = append(failedDns, ips[i])
			}
		}
	} else {
		return http.StatusBadRequest, fmt.Errorf("Could not perform DNS %s lookup. If the proxy is not called 'proxy', you must set SERVICE_NAME=<name-of-the-proxy>.", dns)
	}
	if len(failedDns) > 0 {
		return http.StatusBadRequest, fmt.Errorf("Could not send distribute request to the following addresses: %s", failedDns)
	}
	return http.StatusOK, err
}