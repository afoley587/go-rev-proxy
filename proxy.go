package main

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
)

const (
	ReverseServerAddr    = "0.0.0.0:2002"                // this is our reverse server ip address
	InsideProxyHostname  = "proxy:2002"                  // Requests from private network
	OutsideProxyHostname = "registration.localhost:2002" // Requests from public network
)

// Addresses that have been registered with us
var KnownAddresses = map[string]string{}

func GetScheme(c *gin.Context) string {
	if c.Request.TLS != nil {
		return "https"
	} else {
		return "http"
	}
}

func IsRegistrationRequest(c *gin.Context) bool {
	isRR := ((c.Request.Host == InsideProxyHostname || c.Request.Host == OutsideProxyHostname) &&
		c.Request.URL.String() == "/register")
	return isRR
}

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

func main() {
	r := gin.Default()

	r.Any("/*path", func(c *gin.Context) {

		scheme := GetScheme(c)

		log.Println(scheme, c.Request.Host, c.Request.URL.String())

		if IsRegistrationRequest(c) {

			err := SaveRegistrationRequest(c)

			if err != nil {
				fmt.Println(err)
				c.String(400, "Bad Body")
				return
			}

			c.String(201, "Host Registered")
			return
		}

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
		}

		log.Println("Forwarding request to", remote)

		proxy := httputil.NewSingleHostReverseProxy(remote)

		proxy.Director = func(req *http.Request) {
			req.Header = c.Request.Header
			req.Host = remote.Host
			req.URL.Scheme = remote.Scheme
			req.URL.Host = remote.Host
			req.URL.Path = c.Param("path")
		}

		proxy.ServeHTTP(c.Writer, c.Request)
	})

	if err := r.Run(ReverseServerAddr); err != nil {
		log.Printf("Error: %v", err)
	}
}
