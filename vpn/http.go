package vpn

import (
	"log"
	"math/rand"
	"net/http"
	"net/http/pprof"
)

type httpServer struct {
	*http.Server
	*http.ServeMux
}

func (h httpServer) RegisterHandler(router *router) {
	h.HandleFunc("/debug/ospf", func(w http.ResponseWriter, req *http.Request) {
		ospf := router.routing.Dump()
		_, err := w.Write(ospf)
		if err != nil {
			log.Println(err)
		}
	})
	h.HandleFunc("/debug/speedtest", func(w http.ResponseWriter, req *http.Request) {
		buf := make([]byte, 1024*512)
		rand.Read(buf)
		for {
			_, err := w.Write(buf)
			if err != nil {
				return
			}
		}
	})
	h.HandleFunc("/", func(w http.ResponseWriter, req *http.Request) {
		indexPage := []byte(`<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <title>CuteVPN</title>
</head>
<body>

<a href="/debug/ospf">OSPF</a>
<br/>
<a href="/debug/pprof">net/http/pprof</a>

</body>
</html>`)
		_, err := w.Write(indexPage)
		if err != nil {
			log.Println(err)
		}
	})
}

func startHTTPServer(addr string) httpServer {
	mux := http.NewServeMux()
	server := &http.Server{
		Addr:    addr,
		Handler: mux,
	}
	mux.HandleFunc("/debug/pprof/", pprof.Index)
	mux.HandleFunc("/debug/pprof/cmdline", pprof.Cmdline)
	mux.HandleFunc("/debug/pprof/profile", pprof.Profile)
	mux.HandleFunc("/debug/pprof/symbol", pprof.Symbol)
	mux.HandleFunc("/debug/pprof/trace", pprof.Trace)

	go func() {
		err := server.ListenAndServe()
		if err != http.ErrServerClosed {
			log.Fatal(err)
		}
	}()
	return httpServer{
		Server:   server,
		ServeMux: mux,
	}
}
