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
	"fmt"
	"io/ioutil"
	"net/http"
	"strconv"
)

// consul structs
type ConsulServiceEntry struct {
	Node           string
	Address        string
	ServiceID      string
	ServiceName    string
	ServiceTags    []string
	ServiceAddress string
	ServicePort    int
}

var consulHost = flag.String("h", "", "coul host:port")
var confText bytes.Buffer

func main() {
	flag.Parse()
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
	    mode    http` + "\n\n")

	resMap := consulRes.(map[string]interface{})
	for key, _ := range resMap {
		buildVipConf(key)
	}
	// get all hosts for red/black of each app

	// write config
	fmt.Println(confText.String())
}

func buildVipConf(vipName string) {
	//	fmt.Printf("getting	%s...\n", vipName)
	res, err := http.Get("http://" + *consulHost + "/v1/catalog/service/" + vipName)
	if err != nil {
		fmt.Println("Error getting consul list: ", err)
	}
	defer res.Body.Close()
	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		fmt.Println("Error reading body: ", err)
	}

	var consulRes []ConsulServiceEntry
	err = json.Unmarshal(body, &consulRes)
	if err != nil {
		fmt.Println("Error unmarshaling JSON: ", err)
	}
	//TODO get vip port from consul
	confText.WriteString(`listen ` + vipName + ` 0.0.0.0:8080
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
