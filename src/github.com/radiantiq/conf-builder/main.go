/*
* Copyright 2015 Radiantiq
*
* Licensed under the Apache License, Version 2.0 (the "License");
* you may not use this file except in compliance with the License.
* You may obtain a copy of the License at
*
*     http://www.apache.org/licenses/LICENSE-2.0
*
* Unless required by applicable law or agreed to in writing, software
* distributed under the License is distributed on an "AS IS" BASIS,
* WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
* See the License for the specific language governing permissions and
* limitations under the License.
 */

package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"io/ioutil"
	"log"
	"os"
	"os/signal"
	"sync"
	"syscall"
)

var configFile = flag.String("c", "conf.json", "config file location")
var config = &Conf{}

var confText bytes.Buffer

func main() {
	flag.Parse()
	file, err := ioutil.ReadFile(*configFile)
	if err != nil {
		log.Panic("unable to read config file, exiting...")
	}
	config = &Conf{}
	if err := json.Unmarshal(file, &config); err != nil {
		log.Panic("unable to marshal config file, exiting...")
	}

	stopChan := make(chan bool)
	doneChan := make(chan bool)
	errChan := make(chan error, 10)
	var wg sync.WaitGroup
	watcher := Watcher{StopChan: stopChan, DoneChan: doneChan, ErrorChan: errChan, Waitgroup: wg, Index: 0, Config: *config}

	go watcher.Watch()
	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, syscall.SIGINT, syscall.SIGTERM)
	for {
		select {
		case err := <-errChan:
			log.Println("Error: ", err)
		case s := <-signalChan:
			log.Printf("captured %v exiting...", s)
			close(doneChan)
		case <-doneChan:
			os.Exit(0)
		}
	}
}
