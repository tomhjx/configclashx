package core

import (
	"flag"
	"log"
	"strings"
	"sync"

	"github.com/tomhjx/gogfw"
)

type Processor struct{}

func NewProcessor() *Processor {
	return &Processor{}
}

type Proxy struct {
	Name      string
	Type      string
	Server    string
	Port      uint16
	Password  string
	Sni       string
	AlterId   uint8 `yaml:"alterId"`
	Cipher    string
	Network   string
	Tls       bool
	Uuid      string
	WsHeaders struct {
		Host string `yaml:"Host"`
	} `yaml:"ws-headers"`
	WsPath string `yaml:"ws-path"`
}

type Proxyg struct {
	Name     string
	Type     string
	Url      string
	Interval uint16
	Proxies  []string
}

type stringsFlag []string

func (i *stringsFlag) String() string {
	return strings.Join([]string(*i), ",")
}

func (i *stringsFlag) Set(value string) error {
	*i = append(*i, value)
	return nil
}

func (i *stringsFlag) Get() []string {
	return []string(*i)
}

func addGFWRules(i *target) (res bool, err error) {
	gfwh, err := gogfw.OpenOnline("https://raw.githubusercontent.com/gfwlist/gfwlist/master/gfwlist.txt")
	if err != nil {
		return false, err
	}
	gfwd, err := gfwh.ReadItems()
	if err != nil {
		return false, err
	}
	for _, v := range gfwd {
		rtype := "DOMAIN"
		rval := v.Value
		switch v.Type {
		case gogfw.ITEM_TYPE_DOMAIN, gogfw.ITEM_TYPE_IP:
			rtype = "DOMAIN"
		case gogfw.ITEM_TYPE_DOMAIN_SUFFIX:
			rtype = "DOMAIN-SUFFIX"
		case gogfw.ITEM_TYPE_DOMAIN_KEYWORD:
			rtype = "DOMAIN-KEYWORD"
		}
		i.addRule([]string{rtype, rval, "PROXY"})
	}
	return true, nil
}

func addProxies(t *target, srcp string) (res bool, err error) {

	s, err := OpenOnlineSource(srcp)
	if err != nil {
		log.Println(err)
		return false, err
	}
	proxies, err := s.Proxies()
	if err != nil {
		log.Println(err)
		return false, err
	}
	for _, p := range proxies {
		t.addProxy(p)
	}

	return true, nil
}

func (i *Processor) Run() {
	var (
		srcps stringsFlag
		outp  string
		tplp  string
		help  bool
		wg    sync.WaitGroup
	)
	flag.Var(&srcps, "s", "source's clashx configuration yaml url.")
	flag.StringVar(&outp, "o", "/work/out/clashx.yaml", "output clashx configuration yaml file path.")
	flag.StringVar(&tplp, "tpl", "/work/resources/tpl.yaml", "templet clashx configuration yaml file path.")
	flag.BoolVar(&help, "h", false, "this help")

	flag.Parse()

	if help {
		flag.Usage()
		return
	}

	t, err := newTarget(tplp)
	if err != nil {
		log.Panic(err)
	}

	for _, srcp := range srcps {
		wg.Add(1)
		go func(t *target, srcp string) {
			defer wg.Done()
			addProxies(t, srcp)
		}(t, srcp)
	}

	wg.Add(1)
	go func(t *target) {
		defer wg.Done()
		addGFWRules(t)
	}(t)

	wg.Wait()
	t.persist(outp)
}