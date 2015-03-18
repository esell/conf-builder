package main

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os/exec"
	"strconv"
	"sync"
	"syscall"
	"time"
)

type Watcher struct {
	StopChan  chan bool
	DoneChan  chan bool
	ErrorChan chan error
	Waitgroup sync.WaitGroup
	Index     uint64
}

func WatcherBuild(stopChan, doneChan chan bool, errChan chan error) *Watcher {
	var wg sync.WaitGroup
	return &Watcher{stopChan, doneChan, errChan, wg, 0}

}

func (w *Watcher) Watch() {
	defer close(w.DoneChan)
	w.Waitgroup.Add(1)
	go w.watchService()
	w.Waitgroup.Wait()
}

func (w *Watcher) watchService() {
	defer w.Waitgroup.Done()
	for {
		err := w.getServiceIndex()
		if err != nil {
			w.ErrorChan <- err
			time.Sleep(time.Second * 2)
			continue
		}

	}
}

func (w *Watcher) getServiceIndex() error {
	respChan := make(chan uint64)
	errorChan := make(chan error)
	go func() {
		res, err := http.Get("http://" + *consulHost + "/v1/catalog/services?index=" + strconv.Itoa(int(w.Index)))

		if err != nil {
			fmt.Println("error getting service: ", err)
		}
		defer res.Body.Close()
		//body, err := ioutil.ReadAll(res.Body)
		//if err != nil {
		//	fmt.Println("Error reading body: ", err)
		//}
		fmt.Printf("headers: %v\n", res.Header)
		consulModIndex, err := strconv.Atoi(res.Header["X-Consul-Index"][0])
		if err != nil {
			fmt.Println("error converting consul index: ", err)
			errorChan <- err
		}
		fmt.Println("consul index is: ", consulModIndex)
		respChan <- uint64(consulModIndex)
	}()

	for {
		select {
		case <-w.StopChan:
			return nil
		case e := <-errorChan:
			return e
		case index := <-respChan:
			buildConfig()
			w.Index = index
			return nil
		}
	}

	return nil
}

func buildConfig() {
	res, err := http.Get("http://" + *consulHost + "/v1/catalog/services")
	if err != nil {
		fmt.Println("Error getting consul list: ", err)
	}
	defer res.Body.Close()
	body, err := ioutil.ReadAll(res.Body)
	fmt.Printf("app query: %s\n", body)
	if err != nil {
		fmt.Println("Error reading body: ", err)
	}
	var consulRes interface{}
	err = json.Unmarshal(body, &consulRes)
	if err != nil {
		fmt.Println("Error unmarshaling JSON: ", err)
	}

	confText.WriteString(`defaults
					          log     global
				              mode    http
				              contimeout 5000
				              clitimeout 50000
				              srvtimeout 50000` + "\n\n")

	resMap := consulRes.(map[string]interface{})
	for key, _ := range resMap {
		buildVipConf(key)
	}
	// get all hosts for red/black of each app

	// write config
	fmt.Println(confText.String())
	if err := ioutil.WriteFile("/tmp/cb.out", confText.Bytes(), 0644); err != nil {
		fmt.Println("Unable to write temp config file: ", err)
	}

	err = exec.Command("diff", "/etc/haproxy/haproxy.cfg", "/tmp/cb.out").Run()
	if err != nil {
		if msg, ok := err.(*exec.ExitError); ok {
			fmt.Printf("exit code: %v\n", msg.Sys().(syscall.WaitStatus).ExitStatus())
			if msg.Sys().(syscall.WaitStatus).ExitStatus() == 1 {
				copyAndRestart()
			}
		}

	}
}

func buildVipConf(vipName string) {
	//  fmt.Printf("getting %s...\n", vipName)
	// get haproxy port info
	var haport string
	res, err := http.Get("http://" + *consulHost + "/v1/kv/haportinfo/" + vipName)
	if err != nil {
		fmt.Println("Error getting consul list: ", err)
	}
	defer res.Body.Close()
	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		fmt.Println("Error reading body: ", err)
	}

	var consulPortRes []ConsulEntry
	err = json.Unmarshal(body, &consulPortRes)
	if err != nil {
		fmt.Println("Error unmarshaling JSON: ", err)
	}
	if len(consulPortRes) > 0 {
		if consulPortRes[0].Value == "" {
			haport = "666"
		} else {
			haportByte, err := base64.StdEncoding.DecodeString(consulPortRes[0].Value)
			if err != nil {
				fmt.Println("Error converting base64 value: ", err)
			}
			haport = string(haportByte)
		}
	} else {
		haport = "666"
	}
	res, err = http.Get("http://" + *consulHost + "/v1/catalog/service/" + vipName)
	if err != nil {
		fmt.Println("Error getting consul list: ", err)
	}
	defer res.Body.Close()
	body, err = ioutil.ReadAll(res.Body)
	if err != nil {
		fmt.Println("Error reading body: ", err)
	}

	var consulRes []ConsulServiceEntry
	err = json.Unmarshal(body, &consulRes)
	if err != nil {
		fmt.Println("Error unmarshaling JSON: ", err)
	}
	//TODO get vip port from consul
	confText.WriteString(`listen ` + vipName + ` 0.0.0.0:` + haport + `
                    mode http
                    stats enable
                    stats uri /haproxy?stats
                    balance roundrobin
                    option httpclose
                    option forwardfor
                    `)

	for _, entry := range consulRes {
		confText.WriteString("server " + entry.Node + " " + entry.Address + ":" + strconv.Itoa(entry.ServicePort) + " check\n")
	}
}

func copyAndRestart() {
	cmd := exec.Command("mv", "/tmp/cb.out", "/etc/haproxy/haproxy.cfg")
	if err := cmd.Run(); err != nil {
		fmt.Println("unable to copy new haproxy config ", err)
	}

	cmd = exec.Command("service", "haproxy", "reload")
	if err := cmd.Run(); err != nil {
		fmt.Println("unable to copy new haproxy config ", err)
	}
}
