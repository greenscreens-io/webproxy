package main

import (
	"fmt"
	"log"
	"bytes"
	"flag"
	"strings"
	"net/http"
	"io/ioutil"
	"crypto/tls"
	"crypto/x509"
	"github.com/elazarl/goproxy"
	"github.com/fsnotify/fsnotify"
	"github.com/dop251/goja"
	"github.com/dop251/goja_nodejs/console"
	"github.com/dop251/goja_nodejs/require"
)

type bodyResponseFn func(*string) string
type requestFn func(interface{}) bool

const requestFile = "request.js"
const responseFile = "response.js"

var registry = new(require.Registry)

// default values of lodaded script  is not compiled
var scriptRequest = "function onRequest(req) { return true;}"
var scriptResponse = "function onBodyResponse(bodyContent) { return bodyContent;}"

// handle web page content
func onPageContent(r *http.Response, ctx *goproxy.ProxyCtx) *http.Response {

	if r.ContentLength == 0 {
		return r
	}

	readBody, err := ioutil.ReadAll(r.Body)
	if err != nil {
		log.Println(err)
		return r
	}
	r.Body.Close()

	if len(readBody) > 0 {
		body := string(readBody)
		var onBodyResponse bodyResponseFn
		_, err := initResponseScript(&onBodyResponse)
		if err != nil {
			log.Println(err)
		}

		if err == nil {
			body = onBodyResponse(&body)
			readBody = []byte(body)
		}
	}

	r.Body = ioutil.NopCloser(bytes.NewReader(readBody))
	return r
}

// handle all requests from browser
func requestHandler(r *http.Request, ctx *goproxy.ProxyCtx) (*http.Request, *http.Response) {

	var onRequest requestFn
	_, err := initRequestScript(&onRequest)
	if err != nil {
		log.Println(err)
	}

	if err == nil {
		// TODO here we can add http header modifications

		allowed := onRequest(r)
		if allowed {
			return r, nil
		}
	}

	return r, goproxy.NewResponse(r,
		goproxy.ContentTypeText,
		http.StatusForbidden,
		"Blocked by proxy!")

}

// handle all responses from remote
func responseHandler(r *http.Response, ctx *goproxy.ProxyCtx) *http.Response {

	if r.ContentLength == 0 {
		return r
	}

	ctype := r.Header.Get("Content-Type")
	isHTML := strings.Contains(ctype, "text/html")

	if isHTML {
		return onPageContent(r, ctx)
	}

	return r
}

// initVM initialize JavaScript engine and comppile initial script
func initVM(script string) (*goja.Runtime, error) {
	vm := goja.New()
	registry.Enable(vm)
	console.Enable(vm)
	_, err := vm.RunString(script)
	return vm, err
}

// initRequestScript prepare scripting engine for local requests
func initRequestScript(reqFn *requestFn) (*goja.Runtime, error) {

	vm, err := initVM(scriptRequest)
	if err != nil {
		return nil, err
	}

	funcName := "onRequest"
	_, ok := goja.AssertFunction(vm.Get(funcName))
	if !ok {
		return nil, fmt.Errorf("Function %s not found in script", funcName)
	}

	if reqFn == nil {
		return vm, nil
	}

	err = vm.ExportTo(vm.Get(funcName), reqFn)
	if err != nil {
		return nil, err
	}

	return vm, nil
}

// initResponseScript prepare scripting engine for remote data reponse
func initResponseScript(resFn *bodyResponseFn) (*goja.Runtime, error) {

	vm, err := initVM(scriptResponse)
	if err != nil {
		return nil, err
	}

	funcName := "onBodyResponse"
	_, ok := goja.AssertFunction(vm.Get(funcName))
	if !ok {
		return nil, fmt.Errorf("Function %s not found in script", funcName)
	}

	if resFn == nil {
		return vm, nil
	}

	err = vm.ExportTo(vm.Get(funcName), resFn)
	if err != nil {
		return nil, err
	}

	return vm, nil

}

// loadScript loades initial sources
func loadScript(file string, previous string) (string, error) {
	source, err := ioutil.ReadFile(file)
	if err != nil {
		return previous, err
	}
	return string(source), nil
}

func loadRequestScript() {

	script, err := loadScript(requestFile, scriptRequest)

	if err == nil {
		_, err = initRequestScript(nil)
	}

	if err == nil {
		scriptRequest = script
	}

	if err != nil {
		log.Printf("%s script loading error: %s", requestFile, err)
	}
}

func loadResponseScript() {

	script, err := loadScript(responseFile, scriptResponse)

	if err == nil {
		_, err = initResponseScript(nil)
	}

	if err == nil {
		scriptResponse = script
	}

	if err != nil {
		log.Printf("%s script loading error: %s", responseFile, err)
	}
}

// onWatchFiles triggered when file changed
// will reload into memory for next filter usage
func onWatchFiles(watcher *fsnotify.Watcher) {
	for {
		select {
		case event, ok := <-watcher.Events:
			if !ok {
				return
			}
			if event.Op&fsnotify.Write == fsnotify.Write {
				log.Println("Reloading: " + event.Name)

				if event.Name == requestFile {
					loadRequestScript()
				}

				if event.Name == responseFile {
					loadResponseScript()
				}
			}
		case err, ok := <-watcher.Errors:
			if !ok {
				return
			}
			log.Println("error:", err)
		}
	}
}

// watchFile add a afile to change watcher
func watchFile(watcher *fsnotify.Watcher, file string) {
	err := watcher.Add(file)
	if err != nil {
		panic(err)
	}
}

// watchFiles start watching two root script files
func watchFiles() (watcher *fsnotify.Watcher) {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		panic(err)
	}
	go onWatchFiles(watcher)
	watchFile(watcher, requestFile)
	watchFile(watcher, responseFile)
	return watcher
}

// setCA initialize custom ROOT certificate
func setCA(caCert, caKey []byte) error {
	goproxyCa, err := tls.X509KeyPair(caCert, caKey)
	if err != nil {
		return err
	}
	if goproxyCa.Leaf, err = x509.ParseCertificate(goproxyCa.Certificate[0]); err != nil {
		return err
	}
	goproxy.GoproxyCa = goproxyCa
	goproxy.OkConnect = &goproxy.ConnectAction{Action: goproxy.ConnectAccept, TLSConfig: goproxy.TLSConfigFromCA(&goproxyCa)}
	goproxy.MitmConnect = &goproxy.ConnectAction{Action: goproxy.ConnectMitm, TLSConfig: goproxy.TLSConfigFromCA(&goproxyCa)}
	goproxy.HTTPMitmConnect = &goproxy.ConnectAction{Action: goproxy.ConnectHTTPMitm, TLSConfig: goproxy.TLSConfigFromCA(&goproxyCa)}
	goproxy.RejectConnect = &goproxy.ConnectAction{Action: goproxy.ConnectReject, TLSConfig: goproxy.TLSConfigFromCA(&goproxyCa)}
	return nil
}

func main() {
	verbose := flag.Bool("v", false, "log requests to the stdout")
	addr := flag.String("addr", ":8888", "proxy listen address")
	flag.Parse()

	loadRequestScript()
	loadResponseScript()
	watcher := watchFiles()
	defer watcher.Close()

	// Load your own MITM certificate
	// setCA(caCert, caKey)

	proxy := goproxy.NewProxyHttpServer()
	proxy.OnRequest().HandleConnect(goproxy.AlwaysMitm)
	proxy.OnRequest().DoFunc(requestHandler)
	proxy.OnResponse().DoFunc(responseHandler)
	proxy.Verbose = *verbose
	log.Fatal(http.ListenAndServe(*addr, proxy))
}
