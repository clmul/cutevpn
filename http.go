package cutevpn

import (
	"fmt"
	"log"
	"net/http"
	"net/http/pprof"
)

type HTTPServer struct {
	*http.Server
	*http.ServeMux
}

func (h HTTPServer) RegisterHandler(router *router) {
	const gatewayPath = "/debug/gateway/"
	h.HandleFunc(gatewayPath, func(w http.ResponseWriter, req *http.Request) {
		w.Write([]byte(router.gateway.String()))
	})
	h.HandleFunc("/debug/ospf", func(w http.ResponseWriter, req *http.Request) {
		ospf := router.routing.Dump()
		w.Write(ospf)
	})
}

func StartHTTPServer(host string, port int) HTTPServer {
	mux := http.NewServeMux()
	server := &http.Server{
		Addr:    fmt.Sprintf("%v:%v", host, port),
		Handler: mux,
	}
	mux.HandleFunc("/debug/pprof/", pprof.Index)
	mux.HandleFunc("/debug/pprof/cmdline", pprof.Cmdline)
	mux.HandleFunc("/debug/pprof/profile", pprof.Profile)
	mux.HandleFunc("/debug/pprof/symbol", pprof.Symbol)
	mux.HandleFunc("/debug/pprof/trace", pprof.Trace)
	mux.HandleFunc("/", func(w http.ResponseWriter, req *http.Request) {
		w.Write([]byte(`<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <title>CuteVPN</title>
</head>
<body>

<a href="/debug/pprof">net/http/pprof</a>
<br/>
<a href="/debug/ospf">OSPF</a>
<br/>
<a href="/debug/gateway">Gateway</a>

</body>
</html>`))
	})
	go func() {
		err := server.ListenAndServe()
		if err != http.ErrServerClosed {
			log.Fatal(err)
		}
	}()
	return HTTPServer{
		Server:   server,
		ServeMux: mux,
	}
}
