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
	"encoding/base64"
	"log"
	"net"
	"net/http"
	"net/http/httptest"
	"os/exec"
	"strings"
	"testing"
	"time"
)

var frontEndStaticBody = `    option forwardfor
    option http-server-close
        log     global
        mode    http
        option  httplog
        option  dontlognull
        errorfile 400 /etc/haproxy/errors/400.http
        errorfile 403 /etc/haproxy/errors/403.http
        errorfile 408 /etc/haproxy/errors/408.http
        errorfile 500 /etc/haproxy/errors/500.http
        errorfile 502 /etc/haproxy/errors/502.http
        errorfile 503 /etc/haproxy/errors/503.http
        errorfile 504 /etc/haproxy/errors/504.http
    acl network_allowed src -f /etc/haproxy/pingdom.ip
    acl restricted_page path_beg /systemHealth
    acl host_s3_docs hdr(host) -i docs-admin.layered.com
    acl host_apiary_docs hdr(host) -i docs.layered.com
    http-request deny if restricted_page !network_allowed
    reqadd X-Forwarded-Proto:\ https
    use_backend s3_docs_http if host_s3_docs
    use_backend apiary_docs_http if host_apiary_docs
    default_backend backend_api`

var backEndStaticBody = `    option httpchk GET /systemHealth
    http-check expect string "success":true`

func TestGetGlobals(t *testing.T) {
	success, _ := base64.StdEncoding.DecodeString("CWxvZyAvZGV2L2xvZwlsb2NhbDAKCWxvZyAvZGV2L2xvZwlsb2NhbDEgbm90aWNlCgljaHJvb3QgL3Zhci9saWIvaGFwcm94eQoJc3RhdHMgc29ja2V0IC92YXIvbGliL2hhcHJveHkvc3RhdHMgbW9kZSA3NzcgbGV2ZWwgb3BlcmF0b3IKCXN0YXRzIHRpbWVvdXQgMzBzCgl1c2VyIGhhcHJveHkKCWdyb3VwIGhhcHJveHkKCWRhZW1vbgogICAgICAgIGxvZyAxMC4xMDAuMTMyLjIyMyBsb2NhbDIKICAgICAgICBsb2ctc2VuZC1ob3N0bmFtZQoKCSMgRGVmYXVsdCBTU0wgbWF0ZXJpYWwgbG9jYXRpb25zCgljYS1iYXNlIC9ldGMvc3NsL2NlcnRzCgljcnQtYmFzZSAvZXRjL3NzbC9wcml2YXRlCgoJIyBEZWZhdWx0IGNpcGhlcnMgdG8gdXNlIG9uIFNTTC1lbmFibGVkIGxpc3RlbmluZyBzb2NrZXRzLgoJIyBGb3IgbW9yZSBpbmZvcm1hdGlvbiwgc2VlIGNpcGhlcnMoMVNTTCkuCglzc2wtZGVmYXVsdC1iaW5kLWNpcGhlcnMga0VFQ0RIK2FSU0ErQUVTOmtSU0ErQUVTOitBRVMyNTY6UkM0LVNIQToha0VESDohTE9XOiFFWFA6IU1ENTohYU5VTEw6IWVOVUxM")

	mockConf := Conf{ReloadCmd: "stop", VIPs: []string{"test"}, ConsulHostPort: "http://127.0.0.1:12424", ConsulConfigPath: "/apps/haproxy"}
	mockWatcher := Watcher{Index: 0, Config: mockConf}

	s := buildMockServer(false)
	s.Start()

	res, err := mockWatcher.getGlobalConfig()
	if err != nil {
		t.Errorf("TestGetGlobals failure: ", err)
	}

	if string(res) != string(success) {
		t.Errorf("response is: \n %s \n should be: \n %s \n", string(res), string(success))
	}
	s.Close()
	//time.Sleep(2 * time.Second)
}

func TestGetDefaults(t *testing.T) {
	success, _ := base64.StdEncoding.DecodeString("bG9nCWdsb2JhbAp0aW1lb3V0IGNvbm5lY3QgNTAwMAp0aW1lb3V0IGNsaWVudCAgNTAwMDAKdGltZW91dCBzZXJ2ZXIgIDUwMDAw")
	mockConf := Conf{ReloadCmd: "stop", VIPs: []string{"test"}, ConsulHostPort: "http://127.0.0.1:12424", ConsulConfigPath: "/apps/haproxy"}
	mockWatcher := Watcher{Index: 0, Config: mockConf}

	s := buildMockServer(false)
	s.Start()
	res, err := mockWatcher.getDefaultsConfig()
	if err != nil {
		t.Errorf("TestGetDefaults failure: ", err)
	}
	if string(res) != string(success) {
		t.Errorf("response is: \n %s \n should be: \n %s \n", string(res), string(success))
	}
	s.Close()
}

func TestGetFrontendConf(t *testing.T) {

	mockConf := Conf{ReloadCmd: "stop", VIPs: []string{"test"}, ConsulHostPort: "http://127.0.0.1:12424", ConsulConfigPath: "/apps/haproxy"}
	mockWatcher := Watcher{Index: 0, Config: mockConf}

	s := buildMockServer(false)
	s.Start()
	res := mockWatcher.getFrontendConf("test")
	if res.BindOptions != "ssl crt /etc/ssl/private/layered.com.pem no-sslv3" {
		t.Errorf("TestFrontendConf failure, BindOptions does not match")
	}

	if res.ListenPort != "443" {
		t.Errorf("TestFrontendConf failure, ListenPort does not match")
	}

	if res.Mode != "http" {
		t.Errorf("TestFrontendConf failure, Mode does not match")
	}
	if res.StaticConf != frontEndStaticBody {
		t.Errorf("TestFrontendConf failure, StaticConf does not match")
	}

	s.Close()
}

func TestGetBackendConf(t *testing.T) {

	mockConf := Conf{ReloadCmd: "stop", VIPs: []string{"test"}, ConsulHostPort: "http://127.0.0.1:12424", ConsulConfigPath: "/apps/haproxy"}
	mockWatcher := Watcher{Index: 0, Config: mockConf}

	s := buildMockServer(false)
	s.Start()
	res := mockWatcher.getBackendConf("test")
	if res.BalanceType != "roundrobin" {
		t.Errorf("TestBackendConf failure, BalanceType does not match")
	}

	if res.CatalogMapping != "test-staging" {
		t.Errorf("TestBackendConf failure, CatalogMapping does not match")
	}

	if res.Mode != "http" {
		t.Errorf("TestBackendConf failure, Mode does not match")
	}
	if res.StaticConf != backEndStaticBody {
		t.Errorf("TestBackendConf failure, StaticConf does not match")
	}

	if res.ConfigType != "dynamic" {
		t.Errorf("TestBackendConf failure, ConfigType does not match")
	}

	s.Close()
}

func TestGetRestartCmd(t *testing.T) {

	mockConf := Conf{ReloadCmd: "ls haproxy reload", VIPs: []string{"test"}, ConsulHostPort: "127.0.0.1:12424"}
	mockWatcher := Watcher{Index: 0, Config: mockConf}

	s := buildMockServer(false)
	s.Start()
	res := mockWatcher.getRestartCmd()
	filePath, err := exec.LookPath("ls")
	if err != nil {
		t.Errorf("it appears the ls command doesn't exist on your system so this test will fail")
	}
	if res.Path != filePath {
		t.Errorf("TestGetRestartCmd failure, Path does not match")
	}
	if len(res.Args) != 3 {
		t.Errorf("TestGetRestartCmd failure, Args size is not correct")
	}
	if res.Args[0] != "ls" {
		t.Errorf("TestGetRestartCmd failure, Args[0] does not match")
	}
	if res.Args[1] != "haproxy" {
		t.Errorf("TestGetRestartCmd failure, Args[1] does not match")
	}

	s.Close()
}

func TestBuildConfigNoVIP(t *testing.T) {
	global, _ := base64.StdEncoding.DecodeString("CWxvZyAvZGV2L2xvZwlsb2NhbDAKCWxvZyAvZGV2L2xvZwlsb2NhbDEgbm90aWNlCgljaHJvb3QgL3Zhci9saWIvaGFwcm94eQoJc3RhdHMgc29ja2V0IC92YXIvbGliL2hhcHJveHkvc3RhdHMgbW9kZSA3NzcgbGV2ZWwgb3BlcmF0b3IKCXN0YXRzIHRpbWVvdXQgMzBzCgl1c2VyIGhhcHJveHkKCWdyb3VwIGhhcHJveHkKCWRhZW1vbgogICAgICAgIGxvZyAxMC4xMDAuMTMyLjIyMyBsb2NhbDIKICAgICAgICBsb2ctc2VuZC1ob3N0bmFtZQoKCSMgRGVmYXVsdCBTU0wgbWF0ZXJpYWwgbG9jYXRpb25zCgljYS1iYXNlIC9ldGMvc3NsL2NlcnRzCgljcnQtYmFzZSAvZXRjL3NzbC9wcml2YXRlCgoJIyBEZWZhdWx0IGNpcGhlcnMgdG8gdXNlIG9uIFNTTC1lbmFibGVkIGxpc3RlbmluZyBzb2NrZXRzLgoJIyBGb3IgbW9yZSBpbmZvcm1hdGlvbiwgc2VlIGNpcGhlcnMoMVNTTCkuCglzc2wtZGVmYXVsdC1iaW5kLWNpcGhlcnMga0VFQ0RIK2FSU0ErQUVTOmtSU0ErQUVTOitBRVMyNTY6UkM0LVNIQToha0VESDohTE9XOiFFWFA6IU1ENTohYU5VTEw6IWVOVUxM")
	defaults, _ := base64.StdEncoding.DecodeString("bG9nCWdsb2JhbAp0aW1lb3V0IGNvbm5lY3QgNTAwMAp0aW1lb3V0IGNsaWVudCAgNTAwMDAKdGltZW91dCBzZXJ2ZXIgIDUwMDAw")
	success := `global
` + string(global) + `

defaults
` + string(defaults) + `

`

	mockConf := Conf{ReloadCmd: "service haproxy reload", VIPs: []string{"novip"}, ConsulHostPort: "http://127.0.0.1:12424", ConsulConfigPath: "/apps/haproxy"}
	mockWatcher := Watcher{Index: 0, Config: mockConf}

	s := buildMockServer(true)
	s.Start()
	res := mockWatcher.buildConfig()
	if res != nil {
		t.Errorf("TestBuildConfigNoVIP returned an error: %v", res)
	}
	if confText.String() != success {
		t.Errorf("TestBuildConfigNoVIP results do not match")
		t.Errorf("GOT: %v", confText.String())
		t.Errorf("SHOULD BE: %v", success)
	}
	confText.Reset()
	s.Close()
}

func TestBuildConfig(t *testing.T) {
	global, _ := base64.StdEncoding.DecodeString("CWxvZyAvZGV2L2xvZwlsb2NhbDAKCWxvZyAvZGV2L2xvZwlsb2NhbDEgbm90aWNlCgljaHJvb3QgL3Zhci9saWIvaGFwcm94eQoJc3RhdHMgc29ja2V0IC92YXIvbGliL2hhcHJveHkvc3RhdHMgbW9kZSA3NzcgbGV2ZWwgb3BlcmF0b3IKCXN0YXRzIHRpbWVvdXQgMzBzCgl1c2VyIGhhcHJveHkKCWdyb3VwIGhhcHJveHkKCWRhZW1vbgogICAgICAgIGxvZyAxMC4xMDAuMTMyLjIyMyBsb2NhbDIKICAgICAgICBsb2ctc2VuZC1ob3N0bmFtZQoKCSMgRGVmYXVsdCBTU0wgbWF0ZXJpYWwgbG9jYXRpb25zCgljYS1iYXNlIC9ldGMvc3NsL2NlcnRzCgljcnQtYmFzZSAvZXRjL3NzbC9wcml2YXRlCgoJIyBEZWZhdWx0IGNpcGhlcnMgdG8gdXNlIG9uIFNTTC1lbmFibGVkIGxpc3RlbmluZyBzb2NrZXRzLgoJIyBGb3IgbW9yZSBpbmZvcm1hdGlvbiwgc2VlIGNpcGhlcnMoMVNTTCkuCglzc2wtZGVmYXVsdC1iaW5kLWNpcGhlcnMga0VFQ0RIK2FSU0ErQUVTOmtSU0ErQUVTOitBRVMyNTY6UkM0LVNIQToha0VESDohTE9XOiFFWFA6IU1ENTohYU5VTEw6IWVOVUxM")
	defaults, _ := base64.StdEncoding.DecodeString("bG9nCWdsb2JhbAp0aW1lb3V0IGNvbm5lY3QgNTAwMAp0aW1lb3V0IGNsaWVudCAgNTAwMDAKdGltZW91dCBzZXJ2ZXIgIDUwMDAw")
	success := `global
` + string(global) + `

defaults
` + string(defaults) + `

frontend test
mode http
bind 0.0.0.0:443 ssl crt /etc/ssl/private/layered.com.pem no-sslv3
    option forwardfor
    option http-server-close
        log     global
        mode    http
        option  httplog
        option  dontlognull
        errorfile 400 /etc/haproxy/errors/400.http
        errorfile 403 /etc/haproxy/errors/403.http
        errorfile 408 /etc/haproxy/errors/408.http
        errorfile 500 /etc/haproxy/errors/500.http
        errorfile 502 /etc/haproxy/errors/502.http
        errorfile 503 /etc/haproxy/errors/503.http
        errorfile 504 /etc/haproxy/errors/504.http
    acl network_allowed src -f /etc/haproxy/pingdom.ip
    acl restricted_page path_beg /systemHealth
    acl host_s3_docs hdr(host) -i docs-admin.layered.com
    acl host_apiary_docs hdr(host) -i docs.layered.com
    http-request deny if restricted_page !network_allowed
    reqadd X-Forwarded-Proto:\ https
    use_backend s3_docs_http if host_s3_docs
    use_backend apiary_docs_http if host_apiary_docs
    default_backend backend_api
default_backend test-backend

backend test-backend
mode http
balance roundrobin
    option httpchk GET /systemHealth
    http-check expect string "success":true
server f52104961dc6726a65b4b100e9c3f57c3b0060f97a4654b2eee9b2b8ceb00e1d 10.109.192.82:8080 check
server 22c8fe2e391327e0380474c608841783863160cdad50ddc174490688f588537d 10.109.192.76:8080 check


frontend test2
mode http
bind 0.0.0.0:443 ssl crt /etc/ssl/private/layered.com.pem no-sslv3
    option forwardfor
    option http-server-close
        log     global
        mode    http
        option  httplog
        option  dontlognull
        errorfile 400 /etc/haproxy/errors/400.http
        errorfile 403 /etc/haproxy/errors/403.http
        errorfile 408 /etc/haproxy/errors/408.http
        errorfile 500 /etc/haproxy/errors/500.http
        errorfile 502 /etc/haproxy/errors/502.http
        errorfile 503 /etc/haproxy/errors/503.http
        errorfile 504 /etc/haproxy/errors/504.http
    acl network_allowed src -f /etc/haproxy/pingdom.ip
    acl restricted_page path_beg /systemHealth
    acl host_s3_docs hdr(host) -i docs-admin.layered.com
    acl host_apiary_docs hdr(host) -i docs.layered.com
    http-request deny if restricted_page !network_allowed
    reqadd X-Forwarded-Proto:\ https
    use_backend s3_docs_http if host_s3_docs
    use_backend apiary_docs_http if host_apiary_docs
    default_backend backend_api
default_backend test2-backend

backend test2-backend
mode http
balance roundrobin
    option httpchk GET /systemHealth
    http-check expect string "success":true
server f52104961dc6726a65b4b100e9c3f57c3b0060f97a4654b2eee9b2b8ceb00e1d 10.109.192.82:8080 check
server 22c8fe2e391327e0380474c608841783863160cdad50ddc174490688f588537d 10.109.192.76:8080 check


`

	mockConf := Conf{ReloadCmd: "service haproxy reload", VIPs: []string{"test", "test2"}, ConsulHostPort: "http://127.0.0.1:12424", ConsulConfigPath: "/apps/haproxy"}
	mockWatcher := Watcher{Index: 0, Config: mockConf}

	s := buildMockServer(false)
	s.Start()
	res := mockWatcher.buildConfig()
	if res != nil {
		t.Errorf("TestBuildConfig returned an error: %v", res)
	}
	if strings.TrimSpace(confText.String()) != strings.TrimSpace(success) {
		t.Errorf("TestBuildConfig results do not match")
		t.Errorf("GOT: %v", confText.String())
		t.Errorf("SHOULD BE: %v", success)
	}
	confText.Reset()
	s.Close()
}

func buildMockServer(mockFail bool) httptest.Server {
	// Hack for go 1.5 httptest.Server() race condition
	// https://github.com/golang/go/issues/12262
	time.Sleep(20 * time.Millisecond)
	mux := http.NewServeMux()
	mux.HandleFunc("/v1/kv/apps/haproxy/global", handleProxyGlobal)
	mux.HandleFunc("/v1/kv/apps/haproxy/defaults", handleProxyDefaults)
	mux.HandleFunc("/v1/kv/apps/haproxy/frontend/test/bindOptions", handleFrontBindOpts)
	mux.HandleFunc("/v1/kv/apps/haproxy/frontend/test/listenPort", handleFrontListenPort)
	mux.HandleFunc("/v1/kv/apps/haproxy/frontend/test/mode", handleFrontMode)
	mux.HandleFunc("/v1/kv/apps/haproxy/frontend/test/staticConf", handleFrontStaticConf)
	mux.HandleFunc("/v1/kv/apps/haproxy/frontend/test2/bindOptions", handleFrontBindOpts)
	mux.HandleFunc("/v1/kv/apps/haproxy/frontend/test2/listenPort", handleFrontListenPort)
	mux.HandleFunc("/v1/kv/apps/haproxy/frontend/test2/mode", handleFrontMode)
	mux.HandleFunc("/v1/kv/apps/haproxy/frontend/test2/staticConf", handleFrontStaticConf)
	if !mockFail {
		mux.HandleFunc("/v1/kv/apps/haproxy/frontend/", handleFront)
	}

	mux.HandleFunc("/v1/kv/apps/haproxy/backend/test/balance", handleBackBalance)
	mux.HandleFunc("/v1/kv/apps/haproxy/backend/test/catalogMapping", handleBackCatalogMapping)
	mux.HandleFunc("/v1/kv/apps/haproxy/backend/test/mode", handleFrontMode)
	mux.HandleFunc("/v1/kv/apps/haproxy/backend/test/staticConf", handleBackStaticConf)
	mux.HandleFunc("/v1/kv/apps/haproxy/backend/test/type", handleBackType)
	mux.HandleFunc("/v1/kv/apps/haproxy/backend/test2/balance", handleBackBalance)
	mux.HandleFunc("/v1/kv/apps/haproxy/backend/test2/catalogMapping", handleBackCatalogMapping)
	mux.HandleFunc("/v1/kv/apps/haproxy/backend/test2/mode", handleFrontMode)
	mux.HandleFunc("/v1/kv/apps/haproxy/backend/test2/staticConf", handleBackStaticConf)
	mux.HandleFunc("/v1/kv/apps/haproxy/backend/test2/type", handleBackType)
	if !mockFail {
		mux.HandleFunc("/v1/kv/apps/haproxy/backend/", handleBack)
	}
	mux.HandleFunc("/v1/catalog/service/test-staging", handleCatalogService)
	l, err := net.Listen("tcp", "127.0.0.1:12424")

	if err != nil {

	}
	testHTTPServer := httptest.Server{
		Listener: l,
		Config:   &http.Server{Handler: handlerAccessLog(mux)},
	}
	return testHTTPServer
}

func handleProxyGlobal(w http.ResponseWriter, r *http.Request) {
	writeHeaders(w, 200)
	body := `[{"CreateIndex":115762,"ModifyIndex":115762,"LockIndex":0,"Key":"apps/haproxy/global","Flags":0,"Value":"CWxvZyAvZGV2L2xvZwlsb2NhbDAKCWxvZyAvZGV2L2xvZwlsb2NhbDEgbm90aWNlCgljaHJvb3QgL3Zhci9saWIvaGFwcm94eQoJc3RhdHMgc29ja2V0IC92YXIvbGliL2hhcHJveHkvc3RhdHMgbW9kZSA3NzcgbGV2ZWwgb3BlcmF0b3IKCXN0YXRzIHRpbWVvdXQgMzBzCgl1c2VyIGhhcHJveHkKCWdyb3VwIGhhcHJveHkKCWRhZW1vbgogICAgICAgIGxvZyAxMC4xMDAuMTMyLjIyMyBsb2NhbDIKICAgICAgICBsb2ctc2VuZC1ob3N0bmFtZQoKCSMgRGVmYXVsdCBTU0wgbWF0ZXJpYWwgbG9jYXRpb25zCgljYS1iYXNlIC9ldGMvc3NsL2NlcnRzCgljcnQtYmFzZSAvZXRjL3NzbC9wcml2YXRlCgoJIyBEZWZhdWx0IGNpcGhlcnMgdG8gdXNlIG9uIFNTTC1lbmFibGVkIGxpc3RlbmluZyBzb2NrZXRzLgoJIyBGb3IgbW9yZSBpbmZvcm1hdGlvbiwgc2VlIGNpcGhlcnMoMVNTTCkuCglzc2wtZGVmYXVsdC1iaW5kLWNpcGhlcnMga0VFQ0RIK2FSU0ErQUVTOmtSU0ErQUVTOitBRVMyNTY6UkM0LVNIQToha0VESDohTE9XOiFFWFA6IU1ENTohYU5VTEw6IWVOVUxM"}]`
	w.Write([]byte(body))
}

func handleProxyDefaults(w http.ResponseWriter, r *http.Request) {
	writeHeaders(w, 200)
	body := `[{"CreateIndex":115760,"ModifyIndex":115760,"LockIndex":0,"Key":"apps/haproxy/defaults","Flags":0,"Value":"bG9nCWdsb2JhbAp0aW1lb3V0IGNvbm5lY3QgNTAwMAp0aW1lb3V0IGNsaWVudCAgNTAwMDAKdGltZW91dCBzZXJ2ZXIgIDUwMDAw"}]`
	w.Write([]byte(body))
}

// call is sent with ?raw so we just return the raw text
func handleFrontBindOpts(w http.ResponseWriter, r *http.Request) {
	writeHeaders(w, 200)
	body := `ssl crt /etc/ssl/private/layered.com.pem no-sslv3`
	w.Write([]byte(body))
}

func handleFrontListenPort(w http.ResponseWriter, r *http.Request) {
	writeHeaders(w, 200)
	body := `443`
	w.Write([]byte(body))
}

func handleFrontMode(w http.ResponseWriter, r *http.Request) {
	writeHeaders(w, 200)
	body := `http`
	w.Write([]byte(body))
}

func handleFrontStaticConf(w http.ResponseWriter, r *http.Request) {
	writeHeaders(w, 200)
	w.Write([]byte(frontEndStaticBody))
}

func handleFront(w http.ResponseWriter, r *http.Request) {
	writeHeaders(w, 200)
	body := `["apps/haproxy/frontend/test/","apps/haproxy/frontend/test2/"]`
	w.Write([]byte(body))
}

func handleBack(w http.ResponseWriter, r *http.Request) {
	writeHeaders(w, 200)
	body := `["apps/haproxy/backend/test/","apps/haproxy/backend/test2/"]`
	w.Write([]byte(body))
}

func handleBackBalance(w http.ResponseWriter, r *http.Request) {
	writeHeaders(w, 200)
	body := `roundrobin`
	w.Write([]byte(body))
}
func handleBackCatalogMapping(w http.ResponseWriter, r *http.Request) {
	writeHeaders(w, 200)
	body := `test-staging`
	w.Write([]byte(body))
}
func handleBackStaticConf(w http.ResponseWriter, r *http.Request) {
	writeHeaders(w, 200)
	w.Write([]byte(backEndStaticBody))
}
func handleBackType(w http.ResponseWriter, r *http.Request) {
	writeHeaders(w, 200)
	body := `dynamic`
	w.Write([]byte(body))
}
func handleCatalogService(w http.ResponseWriter, r *http.Request) {
	writeHeaders(w, 200)
	body := `[
  {
    "Node": "f52104961dc6726a65b4b100e9c3f57c3b0060f97a4654b2eee9b2b8ceb00e1d",
    "Address": "10.109.192.82",
    "ServiceID": "f52104961dc6726a65b4b100e9c3f57c3b0060f97a4654b2eee9b2b8ceb00e1d",
    "ServiceName": "test-staging",
    "ServiceTags": [],
    "ServiceAddress": "10.109.192.82",
    "ServicePort": 8080
  },
  {
    "Node": "22c8fe2e391327e0380474c608841783863160cdad50ddc174490688f588537d",
    "Address": "10.109.192.76",
    "ServiceID": "22c8fe2e391327e0380474c608841783863160cdad50ddc174490688f588537d",
    "ServiceName": "test-staging",
    "ServiceTags": [],
    "ServiceAddress": "10.109.192.76",
    "ServicePort": 8080
  }
]`
	w.Write([]byte(body))
}

func writeHeaders(w http.ResponseWriter, code int) {
	h := w.Header()
	h.Add("Content-Type", "application/json")
	w.WriteHeader(code)
}

func handlerAccessLog(handler http.Handler) http.Handler {
	logHandler := func(w http.ResponseWriter, r *http.Request) {
		log.Printf("%s \"%s %s\"", r.RemoteAddr, r.Method, r.URL)
		handler.ServeHTTP(w, r)
	}
	return http.HandlerFunc(logHandler)
}
