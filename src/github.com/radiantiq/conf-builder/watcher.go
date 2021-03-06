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
	"crypto/tls"
	"encoding/base64"
	"encoding/json"
	"io/ioutil"
	"log"
	"net/http"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
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
	Config    Conf
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
func getConsulTransport() *http.Client {
	tlsConfig := tls.Config{MaxVersion: tls.VersionTLS11, InsecureSkipVerify: true}
	myTransport := &http.Transport{
		DisableKeepAlives: true,
		TLSClientConfig:   &tlsConfig,
	}
	return &http.Client{Transport: myTransport}
}

func (w *Watcher) getServiceIndex() error {
	//TODO: need to be async?
	// local chans for async GETs
	respChan := make(chan uint64)
	errorChan := make(chan error)
	transClient := getConsulTransport()

	// clear out previous config
	confText.Reset()
	go func() {
		res, err := transClient.Get(w.Config.ConsulHostPort + "/v1/catalog/services?index=" + strconv.Itoa(int(w.Index)))
		defer res.Body.Close()
		if err != nil {
			log.Println("error getting service: ", err)
			errorChan <- err
		} else {
			log.Printf("headers: %v\n", res.Header)
			consulModIndex, err := strconv.Atoi(res.Header["X-Consul-Index"][0])
			if err != nil {
				log.Println("error converting consul index: ", err)
				errorChan <- err
			}
			log.Println("consul index is: ", consulModIndex)
			respChan <- uint64(consulModIndex)
		}
	}()

	for {
		select {
		case <-w.StopChan:
			return nil
		case e := <-errorChan:
			return e
		case index := <-respChan:
			buildErr := w.buildConfig()
			if buildErr != nil {
				return buildErr
			}
			buildErr = w.writeConfig()
			if buildErr != nil {
				return buildErr
			}
			buildErr = w.updateConfig()
			if buildErr != nil {
				return buildErr
			}
			buildErr = w.copyAndRestart()
			if buildErr != nil {
				return buildErr
			}
			w.Index = index
			return nil
		}
	}

	return nil
}

func (w *Watcher) getGlobalConfig() (globalConfig []byte, err error) {
	transClient := getConsulTransport()
	res, err := transClient.Get(w.Config.ConsulHostPort + "/v1/kv" + w.Config.ConsulConfigPath + "/global")
	if err != nil {
		log.Println("error GETing consul value: ", err)
		return nil, err
	}
	defer res.Body.Close()
	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		log.Println("error reading consul response body: ", err)
		return nil, err
	}

	var consulRes []ConsulEntry
	err = json.Unmarshal(body, &consulRes)
	if err != nil {
		log.Println("error unmarshaling consul response: ", err)
		return nil, err
	}
	globalConfig, err = base64.StdEncoding.DecodeString(consulRes[0].Value)
	if err != nil {
		log.Println("error decoding consul value: ", err)
		return nil, err
	}
	return globalConfig, nil
}

func (w *Watcher) getDefaultsConfig() (defaultsConfig []byte, err error) {
	transClient := getConsulTransport()
	res, err := transClient.Get(w.Config.ConsulHostPort + "/v1/kv" + w.Config.ConsulConfigPath + "/defaults")
	if err != nil {
		log.Println("error GETing consul value: ", err)
		return nil, err
	}
	defer res.Body.Close()
	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		log.Println("error reading consul response body: ", err)
		return nil, err
	}

	var consulRes []ConsulEntry
	err = json.Unmarshal(body, &consulRes)
	if err != nil {
		log.Println("error unmarshaling consul response: ", err)
		return nil, err
	}
	defaultsConfig, err = base64.StdEncoding.DecodeString(consulRes[0].Value)
	if err != nil {
		log.Println("error decoding consul value: ", err)
		return nil, err
	}
	return defaultsConfig, nil
}

func (w *Watcher) buildConfig() error {
	transClient := getConsulTransport()
	// get global
	globalConf, err := w.getGlobalConfig()
	if err != nil {
		return err
	}
	confText.WriteString("global\n")
	confText.WriteString(string(globalConf))
	confText.WriteString("\n\n")
	// get defaults
	defaultsConf, err := w.getDefaultsConfig()
	if err != nil {
		return err
	}
	confText.WriteString("defaults\n")
	confText.WriteString(string(defaultsConf))
	confText.WriteString("\n\n")
	// get all VIPs
	res, err := transClient.Get(w.Config.ConsulHostPort + "/v1/kv" + w.Config.ConsulConfigPath + "/backend/?keys&separator=/")
	if err != nil {
		log.Println("Error getting VIP list from consul: ", err)
		// no VIPs returned but we have global/defaults we can write
		return nil
	}
	defer res.Body.Close()
	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		log.Println("Error reading body: ", err)
		// no VIPs returned but we have global/defaults we can write
		return nil
	}
	var consulRes []string
	err = json.Unmarshal(body, &consulRes)
	if err != nil {
		log.Println("Error unmarshaling frontend list JSON: ", err)
		// no VIPs returned but we have global/defaults we can write
		return nil
	}
	// build VIP config
	for _, val := range consulRes {
		if contains(w.Config.VIPs, filepath.Base(val)) {
			log.Println("building ", filepath.Base(val))
			isSuccess := w.buildVipConf(filepath.Base(val))
			if !isSuccess {
				log.Println("Error building VIP config for ", filepath.Base(val))
			}
		}
	}

	return nil
}
func (w *Watcher) writeConfig() error {
	// write config
	log.Println(confText.String())
	if err := ioutil.WriteFile(w.Config.TempFile, confText.Bytes(), 0644); err != nil {
		log.Println("Unable to write temp config file: ", err)
		return err
	}
	return nil
}

func (w *Watcher) updateConfig() error {
	err := exec.Command("diff", w.Config.ConfigFile, w.Config.TempFile).Run()
	if err != nil {
		if msg, ok := err.(*exec.ExitError); ok {
			log.Printf("exit code: %v\n", msg.Sys().(syscall.WaitStatus).ExitStatus())
			if msg.Sys().(syscall.WaitStatus).ExitStatus() == 1 {
				return nil
			}
		}

	}
	return nil
}

func (w *Watcher) getFrontendConf(name string) Frontend {
	// bindOptions
	bindOptions := w.getConsulString("/v1/kv" + w.Config.ConsulConfigPath + "/frontend/" + name + "/bindOptions?raw")
	// listenPort
	listenPort := w.getConsulString("/v1/kv" + w.Config.ConsulConfigPath + "/frontend/" + name + "/listenPort?raw")
	// mode
	mode := w.getConsulString("/v1/kv" + w.Config.ConsulConfigPath + "/frontend/" + name + "/mode?raw")
	// staticConf
	staticConf := w.getConsulString("/v1/kv" + w.Config.ConsulConfigPath + "/frontend/" + name + "/staticConf?raw")

	return Frontend{BindOptions: bindOptions, ListenPort: listenPort, Mode: mode, StaticConf: staticConf}
}

func (w *Watcher) getBackendConf(name string) Backend {
	// balance
	balanceType := w.getConsulString("/v1/kv" + w.Config.ConsulConfigPath + "/backend/" + name + "/balance?raw")
	// catalogMapping
	catalogMapping := w.getConsulString("/v1/kv" + w.Config.ConsulConfigPath + "/backend/" + name + "/catalogMapping?raw")
	// mode
	mode := w.getConsulString("/v1/kv" + w.Config.ConsulConfigPath + "/backend/" + name + "/mode?raw")
	// staticConf
	staticConf := w.getConsulString("/v1/kv" + w.Config.ConsulConfigPath + "/backend/" + name + "/staticConf?raw")
	// type
	configType := w.getConsulString("/v1/kv" + w.Config.ConsulConfigPath + "/backend/" + name + "/type?raw")

	return Backend{BalanceType: balanceType, CatalogMapping: catalogMapping, Mode: mode, StaticConf: staticConf, ConfigType: configType}
}

func (w *Watcher) getConsulString(path string) string {
	transClient := getConsulTransport()
	result := ""
	res, err := transClient.Get(w.Config.ConsulHostPort + path)
	if err != nil {
		log.Println("Error getting consul list: ", err)
		return result
	}
	defer res.Body.Close()
	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		log.Println("Error reading body: ", err)
		return result
	}
	result = string(body)
	return result
}

func (w *Watcher) buildVipConf(vipName string) bool {
	transClient := getConsulTransport()
	frontEndConf := w.getFrontendConf(vipName)
	emptyFrontEnd := Frontend{BindOptions: "", ListenPort: "", Mode: "", StaticConf: ""}
	if frontEndConf != emptyFrontEnd {
		log.Println("getting frontend config for ", vipName)
		confText.WriteString(`frontend ` + vipName)
		if !strings.HasSuffix(vipName, "\n") {
			confText.WriteString("\n")
		}
		confText.WriteString(`mode ` + frontEndConf.Mode)
		if !strings.HasSuffix(frontEndConf.Mode, "\n") {
			confText.WriteString("\n")
		}
		confText.WriteString(`bind 0.0.0.0:` + frontEndConf.ListenPort + ` ` + frontEndConf.BindOptions)
		if !strings.HasSuffix(frontEndConf.BindOptions, "\n") {
			confText.WriteString("\n")
		}
		confText.WriteString(frontEndConf.StaticConf)
		if !strings.HasSuffix(frontEndConf.StaticConf, "\n") {
			confText.WriteString("\n")
		}
		confText.WriteString(`default_backend ` + vipName + `-backend`)
		confText.WriteString("\n\n")
	}
	backEndConf := w.getBackendConf(vipName)
	emptyBackEnd := Backend{BalanceType: "", CatalogMapping: "", Mode: "", StaticConf: "", ConfigType: ""}
	if backEndConf != emptyBackEnd {
		log.Println("getting backend config for ", vipName)
		confText.WriteString(`backend ` + vipName + `-backend`)
		confText.WriteString("\n")
		confText.WriteString(`mode ` + backEndConf.Mode)
		if !strings.HasSuffix(backEndConf.Mode, "\n") {
			confText.WriteString("\n")
		}
		confText.WriteString(`balance ` + backEndConf.BalanceType)
		if !strings.HasSuffix(backEndConf.BalanceType, "\n") {
			confText.WriteString("\n")
		}
		confText.WriteString(backEndConf.StaticConf)
		if !strings.HasSuffix(backEndConf.StaticConf, "\n") {
			confText.WriteString("\n")
		}
		if backEndConf.ConfigType == "dynamic" {
			res, err := transClient.Get(w.Config.ConsulHostPort + "/v1/catalog/service/" + backEndConf.CatalogMapping)
			if err != nil {
				log.Println("Error getting consul list: ", err)
				return false
			}
			defer res.Body.Close()
			body, err := ioutil.ReadAll(res.Body)
			if err != nil {
				log.Println("Error reading body: ", err)
				return false
			}

			var consulRes []ConsulServiceEntry
			err = json.Unmarshal(body, &consulRes)
			if err != nil {
				log.Println("Error unmarshaling service JSON: ", err)
				return false
			}
			for _, entry := range consulRes {
				confText.WriteString("server " + entry.Node + " " + entry.Address + ":" + strconv.Itoa(entry.ServicePort) + " check\n")
			}
		}
		confText.WriteString("\n\n")
	}
	return true
}

func (w *Watcher) copyAndRestart() error {
	cmd := exec.Command("mv", w.Config.TempFile, w.Config.ConfigFile)
	if err := cmd.Run(); err != nil {
		log.Println("unable to copy new haproxy config ", err)
		return err
	}

	cmd = w.getRestartCmd()
	output, err := cmd.CombinedOutput()
	if err != nil {
		log.Println("unable to reload haproxy ", err)
		log.Println("output: ", string(output))
		return err
	}
	log.Println("output: ", string(output))
	return nil
}

func (w *Watcher) getRestartCmd() *exec.Cmd {
	cmdSplits := strings.Split(w.Config.ReloadCmd, " ")
	path, err := exec.LookPath(cmdSplits[0])
	if err != nil {
		log.Printf("unable to find %s on the system\n", cmdSplits[0])
	}
	if len(cmdSplits) == 1 {
		return &exec.Cmd{Path: path}
	}
	returnCmd := exec.Cmd{Path: path, Args: cmdSplits[0:]}
	return &returnCmd
}

func contains(s []string, e string) bool {
	for _, a := range s {
		if a == e {
			return true
		}
	}
	return false
}
