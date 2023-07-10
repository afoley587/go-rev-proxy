# Dynamic Hostname Based Reverse Proxy With GoLang

I use [Traefik](https://traefik.io/traefik/) in production scenarios
and was curious if I could build something similar in a day.
Not the same product, as traefik is feature rich and absolutely amazing, 
but I did want to try to build a reverse proxy with golang that some docker 
containers could register themselves with.

So, first thing is first, what is a reverse proxy? A reverse proxy is a 
server or application that sits in front of other web servers or applications.
It then forwards client requests to those remote web servers. It can act as a 
single entrypoint in to your network, a load balancer, etc.

Reverse proxies can operate on a bunch of different methods for forwarding traffic. Some
of the more popular ones are:

* Hostname-based routing: If the host header in a packet matches some pattern,
	use that as a rule to find and forward to the remote host.
* Path-based routing: If the path in a request matches some pattern,
	use that as a rule to find and forward to the remote host.
* Header-based routing: If some header key/value pair matches some pattern,
	use that as a rule to find and forward to the remote host.

And of course, there can be combinations of the above. In this demo, we will
look at hostname-based routing. This is mainly preference. I love having separate
DNS names for separate apps, instead of separate paths for separate apps. I then
like to use wildcard DNS records for SSL certificates, so even though I may have 10
DNS records, I have one cert and one load balancer, and it just all works!

# The Code

Let's walk through the code. If you want to jump right to the finished product, 
feel free to head right to my [GitHub here](github.comafoley587/go-rev-proxy.git)!

Let's start with a few variables that we will use later in the code:

```golang
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
```

We start with a `RunPort` which is the port we will be listening on 
(for example, `localhost:2002`). We then define the `ReverseServerAddr` which 
will be set to the unspecified address and our `RunPort` (`0.0.0.0:2002`).
So our server will listen both publically and privately on port 2002.
We then define an `InsideProxyHostname` and `OutsideProxyHostname`, which
our reverse server will specifically use to listen for registration requests 
(more on that later!). Finally, we store a map of `KnownAddresses` - or addresses
that have registered with the reverse proxy.

Next, we will define a our `Proxy` function which will do the actual proxying of requests:

```go
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
```

The `Proxy` function is pretty simple and has two main purposes:

1. If this is a registration request, we will register the new inside/outside
	hostname pair and return.
2. Otherwise, try to proxy the request to the inside (internal) server.


We won't go too much in detail about `IsRegistrationRequest` and 
`SaveRegistrationRequest` as they are super straight forward. From a 
high level:

* `IsRegistrationRequest` - Checks the hostname and path to see if it matches
	a set of criteria that we use to deem it a "Registration Request". In 
	our system, we deem it a registration request if:
	* The hostname is either `registration.localhost` or `proxy` AND if the path
		is `/register`.
* `SaveRegistrationRequest` - Gathers the request from the user and saves the 
	inside/outside hostname pairing for further use in our `KnownAddresses`
	map.

If this is not a registration request, we will then just try to forward our 
request along by:

1. Seeing if the outside hostname is in our `KnownAddresses` map
	a. If not, we throw an error and return
2. Creating a new, translated URL. Essentially, we will substitute the 
	outside hostname in the original URL with the internal URL that
	we had previously saved.
3. Creating a new [ReverseProxy](https://pkg.go.dev/net/http/httputil#ReverseProxy) instance
	with the original headers, scheme, and path, but with our translated host. If there is an
	error with our remote server, you would see it here and it would be returned to the requesting
	client.

That's about all there is to it. At this point, you can do hostname based routing with a self-made
reverse proxy.

# Running an example

There is a quick and easy docker-compose file in the GitHub repo. You can find it [here](https://github.com/afoley587/go-rev-proxy/blob/main/examples/docker-compose.yml).
You can run `docker-compose up` and it will bring up three services for you:

1. The proxy service listening on localhost:2002
2. Two NGINX servers which are not listening on public endpoints. They will
	register with the proxy on two domain names:
		1. truck.localhost:2002
		2. car.localhost:2002

Once the stack is up, open a browser and go to 
`truck.localhost:2002` and `car.localhost:2002` and notice how
the hostnames are changing to the hostnames of your
docker containers! 

![demo](./img/demo.gif)

# GitHub
As previously mentioned, feel free to navigate [here](https://github.com/afoley587/go-rev-proxy)
for all rev-proxy code!
