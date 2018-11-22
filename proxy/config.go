package proxy

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"math/rand"
	"os"
	"sync"
)

type Config struct {
	Server     string
	PassWord   string
	RemotePort int
	LocalAddr  string
	LocalPort  int
	Method     string
}

type Configs struct {
	Configs   []Config
	LocalPort int
}

func Load(filename string) Configs {
	var v Configs
	data, err := ioutil.ReadFile(filename)
	if err != nil {
		fmt.Fprintf(os.Stdout, "error occured %v", err)
		os.Exit(0)
	}
	json.Unmarshal(data, &v)
	return v
}

type ProxyRef map[string]int

var syncLock sync.Mutex

func IncRef(mp map[string]ProxyRef, host, proxy string) {
	syncLock.Lock()
	if _, ok := mp[host]; ok {
		if _, ok := mp[host][proxy]; ok {
			mp[host][proxy]++
		} else {
			mp[host][proxy] = 1
		}
	} else {
		mp[host] = make(ProxyRef)
		mp[host][proxy] = 1
	}
	syncLock.Unlock()
}

func DecRef(mp map[string]ProxyRef, host, proxy string) {
	syncLock.Lock()
	defer func() {
		if e := recover(); e != nil {
			fmt.Printf("Panicing %s\r\n", e)
		}
	}()
	if _, ok := mp[host]; ok {
		if _, ok := mp[host][proxy]; ok {
			mp[host][proxy]--
		} else {
			log.Fatalln("Cant't find this proxy from this map")
		}
	} else {
		log.Fatalln("Can't find this host from connection map")
	}
	syncLock.Unlock()
}

func initProxyRef(mp map[string]ProxyRef, proxies []string, domain string) {
	mp[domain] = make(map[string]int)
	for _, p := range proxies {
		mp[domain][p] = 0
	}
}

func FindBestProxy(mp map[string]ProxyRef, proxies []string, domain string) string {
	syncLock.Lock()
	_, ok := mp[domain]
	if !ok {
		initProxyRef(mp, proxies, domain)
	}
	inf := int(^uint(0) >> 1)
	var rst string
	var keys []string
	for key, _ := range mp[domain] {
		keys = append(keys, key)
	}
	rand.Shuffle(len(keys), func(i, j int) {
		keys[i], keys[j] = keys[j], keys[i]
	})
	for _, key := range keys {
		val := mp[domain][key]
		if val < inf {
			inf = val
			rst = key
		}
	}
	syncLock.Unlock()
	return rst
}
