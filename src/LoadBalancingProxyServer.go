package main

import (
	"fmt"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
)

type Server interface {
	Address() string                                 // domain name
	IsAlive() bool                                   // servers is up or down? will reroute the request to other server if this server is down
	Serve(rw http.ResponseWriter, req *http.Request) // serve the request
}

type SimpleServer struct {
	addr  string
	proxy *httputil.ReverseProxy // rerouting the request to somewhere else
}

func (s *SimpleServer) Address() string {
	return s.addr
}

func (s *SimpleServer) IsAlive() bool { // just return true for now
	return true
}

func (s *SimpleServer) Serve(rw http.ResponseWriter, req *http.Request) {
	s.proxy.ServeHTTP(rw, req)
}

func newSimpleServer(addr string) *SimpleServer {
	serverUrl, err := url.Parse(addr)
	handleErr(err)
	return &SimpleServer{
		addr:  addr,
		proxy: httputil.NewSingleHostReverseProxy(serverUrl),
	}
}

type LoadBalancer struct {
	port            string   // port no.
	roundRobinCount int      // which server to reroute to
	servers         []Server // list of servers connected
}

func newLoadBalancer(port string, servers []Server) *LoadBalancer {
	return &LoadBalancer{
		port:            port,
		roundRobinCount: 0,
		servers:         servers,
	}
}

func (lb *LoadBalancer) getNextAvailableServer() Server {
	// basic roundrobin using mod operator
	server := lb.servers[lb.roundRobinCount%len(lb.servers)]
	for !server.IsAlive() { // if the server is down, get the next live server
		lb.roundRobinCount++
		server = lb.servers[lb.roundRobinCount%len(lb.servers)]
	}
	lb.roundRobinCount++
	return server
}

func (lb *LoadBalancer) serveProxy(rw http.ResponseWriter, req *http.Request) {
	targetServer := lb.getNextAvailableServer()
	fmt.Printf("forwarding request to address %q\n", targetServer.Address())
	targetServer.Serve(rw, req)
}

func handleErr(err error) {
	if err != nil {
		fmt.Printf("error: %v\n", err)
		os.Exit(1)
	}
}

func main() {
	// list of servers to be connected to load balancing proxy server
	servers := []Server{
		newSimpleServer("https://www.github.com"),
		newSimpleServer("https://www.amazon.com"),
		newSimpleServer("https://www.medium.com"),
	}

	lb := newLoadBalancer("8001", servers) // load balancing proxy server runs on port 8001 on localhost

	//
	handleRedirect := func(rw http.ResponseWriter, req *http.Request) {
		lb.serveProxy(rw, req)
	}

	http.HandleFunc("/", handleRedirect)
	fmt.Printf("serving requests at 'localhost:%s'\n", lb.port)
	http.ListenAndServe(":"+lb.port, nil)
}
