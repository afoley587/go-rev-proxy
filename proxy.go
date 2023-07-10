package main

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
)

var (
	RunPort              = 2002                                           // The server port to run on
	ReverseServerAddr    = fmt.Sprint("0.0.0.0:", RunPort)                // this is our reverse server ip address
	InsideProxyHostname  = fmt.Sprint("proxy:", RunPort)                  // Requests from private network
	OutsideProxyHostname = fmt.Sprint("registration.localhost:", RunPort) // Requests from public network
	KnownAddresses       = map[string]string{}                            // Known Addresses
)

// This function checks if TLS was enabled on the request
// and translates it to the proper scheme (http or https)
func GetScheme(c *gin.Context) string {
	if c.Request.TLS != nil {
		return "https"
	} else {
		return "http"
	}
}

// IsRegistrationRequest checks if an incoming request is meant to register
// a new inside/outside hostname pair by checking if the hostname and
// path match a specific combination.
// If the host is our registration.localhost or proxy hostname
// AND if the path is /register, we will register a new endpoint
func IsRegistrationRequest(c *gin.Context) bool {
	isRR := ((c.Request.Host == InsideProxyHostname || c.Request.Host == OutsideProxyHostname) &&
		c.Request.URL.String() == "/register")
	return isRR
}

// SaveRegistrationRequest saves an inside/outside hostname pairing
// to our KnownAddresses map so we can use it later. Effectively
// registering a new endpoint for us
func SaveRegistrationRequest(c *gin.Context) error {
	var rr RegistrationRequest
	log.Println(c.Request.Body)
	err := c.BindJSON(&rr)

	if err != nil {
		return err
	}

	log.Println("Registering", rr.OutsideHost, "to", rr.InsideHost)
	KnownAddresses[rr.OutsideHost] = rr.InsideHost
	return nil
}

// Proxy runs the actual proxy and will look at the
// hostnames requested from the received request. It will
// then translate that to the inside hostname and forward the
// request
func Proxy(c *gin.Context) {

	// Get if HTTP or HTTPS
	scheme := GetScheme(c)

	log.Println(scheme, c.Request.Host, c.Request.URL.String())

	// If this is a registration request, save it and
	// then stop processing this request
	if IsRegistrationRequest(c) {

		err := SaveRegistrationRequest(c)

		if err != nil {
			log.Println(err)
			c.String(400, "Couldnt Register Host")
			return
		}

		c.String(201, "Host Registered")
		return
	}

	// Translate the outside hostname to the inside hostname
	forwardTo, ok := KnownAddresses[c.Request.Host]

	if !ok {
		log.Printf("Unkown Host: %v", c.Request.Host)
		c.String(400, "Unkown Host")
		return
	}

	rUrl := fmt.Sprintf("%v://%v%v", scheme, forwardTo, c.Request.URL)

	remote, err := url.Parse(rUrl)

	if err != nil {
		log.Println(err)
		c.String(500, "Error Proxying Host")
		return
	}

	log.Println("Forwarding request to", remote)

	// Forward the request to the inside remote server
	// https://pkg.go.dev/net/http/httputil#NewSingleHostReverseProxy
	proxy := httputil.NewSingleHostReverseProxy(remote)

	// Director is a function which modifies
	// the request into a new request to be sent
	// https://pkg.go.dev/net/http/httputil#ReverseProxy
	proxy.Director = func(req *http.Request) {
		req.Header = c.Request.Header
		req.Host = remote.Host
		req.URL.Scheme = remote.Scheme
		req.URL.Host = remote.Host
		req.URL.Path = c.Param("path")
	}

	proxy.ServeHTTP(c.Writer, c.Request)
}

func main() {
	r := gin.Default()

	// all paths should be handled by Proxy()
	r.Any("/*path", Proxy)

	if err := r.Run(ReverseServerAddr); err != nil {
		log.Printf("Error: %v", err)
	}
}
