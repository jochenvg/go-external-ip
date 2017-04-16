package externalip

import (
	"io/ioutil"
	"net"
	"net/http"
	"strings"

	"github.com/miekg/dns"
)

var queriers = []func() string{
	GoogleDNS,
	OpenDNS,
	AkamaiDNS,
}

// DNS queries all DNS based external IP resolvers in parallel and
// produces a result if there is a quorum - at least half of the results
// are the same.
func DNS() string {
	// Run the DNS queries in a goroutine, channeling the results into ch
	ch := make(chan string)
	for _, f := range queriers {
		go func(f func() string) {
			ch <- f()
		}(f)
	}
	// Receive the results from the queries from ch and produce a result
	// if more than half of them concur.
	result := make(chan string)
	go func() {
		ips := make(map[string]int)
		done := false
		for range queriers {
			ip := <-ch
			ips[ip]++
			if !done && ips[ip] > len(queriers)/2 {
				result <- ip
				done = true
			}
		}
		close(result)
	}()
	return <-result
}

// GoogleDNS queries Google Public DNS for the external IP address
func GoogleDNS() (ip string) {
	msg := new(dns.Msg)
	msg.SetQuestion("o-o.myaddr.l.google.com.", dns.TypeTXT)
	in, err := dns.Exchange(msg, "ns1.google.com:53")
	if err != nil {
		return
	}
	if t, ok := in.Answer[0].(*dns.TXT); ok {
		ip = net.ParseIP(t.Txt[0]).To4().String()
	}
	return
}

// OpenDNS queries Open DNS for the external IP address
func OpenDNS() (ip string) {
	msg := new(dns.Msg)
	msg.SetQuestion("myip.opendns.com.", dns.TypeA)
	in, err := dns.Exchange(msg, "resolver1.opendns.com:53")
	if err != nil {
		return
	}
	if a, ok := in.Answer[0].(*dns.A); ok {
		ip = a.A.To4().String()
	}
	return
}

// AkamaiDNS queries Akamai DNS for the external IP address
func AkamaiDNS() (ip string) {
	msg := new(dns.Msg)
	msg.SetQuestion("whoami.akamai.net.", dns.TypeA)
	in, err := dns.Exchange(msg, "ns1-1.akamaitech.net:53")
	if err != nil {
		return
	}
	if a, ok := in.Answer[0].(*dns.A); ok {
		ip = a.A.To4().String()
	}
	return
}

var urls = []string{
	"http://v4.ident.me/",
	"http://whatismyip.akamai.com/",
	"http://checkip.amazonaws.com/",
	"http://ipecho.net/plain",
	"http://inet-ip.info/ip",
	"http://eth0.me/",
	"http://wgetip.com/",
	"http://bot.whatismyipaddress.com/",
	"http://ipof.in/txt",
	"http://smart-ip.net/myip",
	"https://ip.tyk.nu/",
	"https://tnx.nl/ip",
	"https://l2.io/ip",
	"https://api.ipify.org/",
	"https://myexternalip.com/raw",
	"https://icanhazip.com",
	"https://ifconfig.io/ip",
	"https://wtfismyip.com/text",
}

// HTTP queries all HTTP based external IP resolvers in parallel and
// produces a result if there is a quorum - at least half of the results
// are the same.
func HTTP() string {
	// Run the web queries in a goroutine, channeling the result into ch
	ch := make(chan string)
	for _, url := range urls {
		go func(url string) {
			ch <- urlGetReadAll(url)
		}(url)
	}
	result := make(chan string)
	go func() {
		ips := make(map[string]int)
		done := false
		for range urls {
			ip := <-ch
			ips[ip]++
			if !done && ips[ip] > len(urls)/2 {
				result <- ip
				done = true
			}
		}
		close(result)
	}()
	return <-result
}

func urlGetReadAll(url string) (ip string) {
	resp, err := http.Get(url)
	if err != nil {
		return
	}
	defer resp.Body.Close()

	bytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return
	}
	return strings.TrimSpace(string(bytes))
}
