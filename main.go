package main

import (
	"fmt"
	"math/rand"
	"net"
	"time"

	"github.com/joho/godotenv"
	"github.com/miekg/dns"
	"github.com/qiangxue/go-env"
)

type donotshoutServer struct {
	Host            string
	Port            int16
	Protocol        string
	MinJitter       int32
	MaxJitter       int32
	IPv4Address     string
	IPv6Address     string
	TruncatePercent int
	DropPercent     int
}

var chaos *rand.Rand

func chaosRanged(bottom, top int32) int32 {
	return bottom + (chaos.Int31() % top)
}

func chaosDo(chance int) bool {
	return chaos.Int()%100 < chance
}

func (srv *donotshoutServer) ServeDNS(rw dns.ResponseWriter, r *dns.Msg) {
	for _, q := range r.Question {
		if q.Qtype == dns.TypeA {
			fmt.Printf("A QUERY=%+v\n", q)
			r := &dns.Msg{
				MsgHdr: dns.MsgHdr{
					Id:            r.Id,
					Response:      true,
					Authoritative: true,
				},
			}
			r.Question = append(r.Question, q)
			r.Answer = append(r.Answer, &dns.A{
				Hdr: dns.RR_Header{
					Name:   q.Name,
					Rrtype: dns.TypeA,
					Class:  dns.ClassINET,
					Ttl:    1,
				},
				A: net.ParseIP(srv.IPv4Address),
			})
			data, _ := r.Pack()

			if chaosDo(srv.DropPercent) == true {
				fmt.Println("dropped packet")
				return
			}

			if chaosDo(srv.TruncatePercent) == true {
				fmt.Println("truncated packet")
				// allow for at least the header?
				data = data[0:chaosRanged(1, int32(len(data)-1))]
			}

			jitter := chaosRanged(srv.MinJitter, srv.MaxJitter)
			if jitter >= 1000 {
				fmt.Printf("jitter=%d\n", jitter)
			}
			td := time.NewTimer(time.Millisecond * time.Duration(jitter))
			<-td.C

			rw.Write(data)

		} else if q.Qtype == dns.TypeAAAA {
			fmt.Printf("AAAA QUERY=%+v\n", q)
			r := &dns.Msg{
				MsgHdr: dns.MsgHdr{
					Id:            r.Id,
					Response:      true,
					Authoritative: true,
				},
			}

			r.Question = append(r.Question, q)
			r.Answer = append(r.Answer, &dns.AAAA{
				Hdr: dns.RR_Header{
					Name:   q.Name,
					Rrtype: dns.TypeAAAA,
					Class:  dns.ClassINET,
					Ttl:    1,
				},
				AAAA: net.ParseIP(srv.IPv6Address),
			})

			data, _ := r.Pack()

			jitter := chaosRanged(srv.MinJitter, srv.MaxJitter)
			if jitter >= 1000 {
				fmt.Printf("jitter=%d\n", jitter)
			}
			td := time.NewTimer(time.Millisecond * time.Duration(jitter))
			<-td.C

			if chaosDo(srv.DropPercent) == true {
				fmt.Println("dropped packet")
				return
			}

			if chaosDo(srv.TruncatePercent) == true {
				fmt.Println("truncated packet")
				// allow for at least the header?
				data = data[0:chaosRanged(1, int32(len(data)-1))]
			}

			rw.Write(data)
		}
	}
}

func main() {
	godotenv.Load()
	chaos = rand.New(rand.NewSource(time.Now().Unix()))
	// Default
	srv := &donotshoutServer{
		Host:            "0.0.0.0",
		Port:            53,
		IPv4Address:     "127.0.0.1",
		IPv6Address:     "::1",
		Protocol:        "udp",
		MinJitter:       1,
		MaxJitter:       5000,
		TruncatePercent: 10,
		DropPercent:     5,
	}
	loader := env.New("", nil)
	if err := loader.Load(srv); err != nil {
		panic(err)
	}
	err := dns.ListenAndServe(fmt.Sprintf("%s:%d", srv.Host, srv.Port),
		srv.Protocol, srv)
	if err != nil {
		panic(err)
	}
}
