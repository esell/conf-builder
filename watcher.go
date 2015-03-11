package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strconv"
	"sync"
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
}
