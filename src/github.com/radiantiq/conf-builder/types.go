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

type ConsulEntry struct {
	CreateIndex int64  `json:"CreateIndex"`
	ModifyIndex int64  `json:"ModifyIndex"`
	LockIndex   int64  `json:"LockIndex"`
	Key         string `json:"Key"`
	Flags       int64  `json:"Flags"`
	Value       string `json:"Value"`
}

type ConsulServiceEntry struct {
	Node           string
	Address        string
	ServiceID      string
	ServiceName    string
	ServiceTags    []string
	ServiceAddress string
	ServicePort    int
}

type Conf struct {
	ReloadCmd        string   `json:"haproxyReloadCmd"`
	VIPs             []string `json:"vips"`
	ConsulHostPort   string   `json:"consulHostPort"`
	ConfigFile       string   `json:"configFile"`
	TempFile         string   `json:"tempFile"`
	ConsulConfigPath string   `json:"consulConfigPath"`
}

type Frontend struct {
	BindOptions string
	ListenPort  string
	Mode        string
	StaticConf  string
}

type Backend struct {
	BalanceType    string
	CatalogMapping string
	Mode           string
	StaticConf     string
	ConfigType     string
}
