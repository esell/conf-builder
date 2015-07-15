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
	"log"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestGetGlobals(t *testing.T) {
	success := `	log /dev/log	local0
	log /dev/log	local1 notice
	chroot /var/lib/haproxy
	stats socket /var/lib/haproxy/stats mode 777 level operator
	stats timeout 30s
	user haproxy
	group haproxy
	daemon
        log 10.100.132.223 local2
        log-send-hostname

	# Default SSL material locations
	ca-base /etc/ssl/certs
	crt-base /etc/ssl/private

	# Default ciphers to use on SSL-enabled listening sockets.
	# For more information, see ciphers(1SSL).
	ssl-default-bind-ciphers kEECDH+aRSA+AES:kRSA+AES:+AES256:RC4-SHA:!kEDH:!LOW:!EXP:!MD5:!aNULL:!eNULL`

	mockConf := Conf{ReloadCmd: "stop", VIPs: []string{"test"}, ConsulHostPort: "127.0.0.1:12424"}
	mockWatcher := Watcher{Index: 0, Config: mockConf}

	s := buildMockServer()
	s.Start()

	res, err := mockWatcher.getGlobalConfig()
	if err != nil {
		t.Errorf("TestGetGlobals failure: ", err)
	}

	if string(res) != success {
		t.Errorf("response does not match")
	}
	s.Close()
}

func TestGetDefaults(t *testing.T) {
	success := `log	global
timeout connect 5000
timeout client  50000
timeout server  50000`

	mockConf := Conf{ReloadCmd: "stop", VIPs: []string{"test"}, ConsulHostPort: "127.0.0.1:12424"}
	mockWatcher := Watcher{Index: 0, Config: mockConf}

	s := buildMockServer()
	s.Start()
	res, err := mockWatcher.getDefaultsConfig()
	if err != nil {
		t.Errorf("TestGetDefaults failure: ", err)
	}

	if string(res) != success {
		t.Errorf("response does not match")
	}
	s.Close()
}

func TestGetFrontendConf(t *testing.T) {

	mockConf := Conf{ReloadCmd: "stop", VIPs: []string{"test"}, ConsulHostPort: "127.0.0.1:12424"}
	mockWatcher := Watcher{Index: 0, Config: mockConf}

	s := buildMockServer()
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
	staticConf := `    option forwardfor
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
	if res.StaticConf != staticConf {
		t.Errorf("TestFrontendConf failure, StaticConf does not match")
	}

	s.Close()
}

func TestGetBackendConf(t *testing.T) {

	mockConf := Conf{ReloadCmd: "stop", VIPs: []string{"test"}, ConsulHostPort: "127.0.0.1:12424"}
	mockWatcher := Watcher{Index: 0, Config: mockConf}

	s := buildMockServer()
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
	staticConf := `    option httpchk GET /systemHealth
    http-check expect string "success":true`
	if res.StaticConf != staticConf {
		t.Errorf("TestBackendConf failure, StaticConf does not match")
	}

	if res.ConfigType != "dynamic" {
		t.Errorf("TestBackendConf failure, ConfigType does not match")
	}

	s.Close()
}

func TestGetRestartCmd(t *testing.T) {

	mockConf := Conf{ReloadCmd: "service haproxy reload", VIPs: []string{"test"}, ConsulHostPort: "127.0.0.1:12424"}
	mockWatcher := Watcher{Index: 0, Config: mockConf}

	s := buildMockServer()
	s.Start()
	res := mockWatcher.getRestartCmd()
	if res.Path != "service" {
		t.Errorf("TestGetRestartCmd failure, Path does not match")
	}
	if len(res.Args) != 2 {
		t.Errorf("TestGetRestartCmd failure, Args size is not correct")
	}
	if res.Args[0] != "haproxy" {
		t.Errorf("TestGetRestartCmd failure, Args[0] does not match")
	}
	if res.Args[1] != "reload" {
		t.Errorf("TestGetRestartCmd failure, Args[1] does not match")
	}

	s.Close()
}

func buildMockServer() *httptest.Server {
	mux := http.NewServeMux()
	mux.HandleFunc("/v1/kv/apps/haproxy/global", handleProxyGlobal)
	mux.HandleFunc("/v1/kv/apps/haproxy/defaults", handleProxyDefaults)
	mux.HandleFunc("/v1/kv/apps/haproxy/frontend/test/bindOptions", handleFrontBindOpts)
	mux.HandleFunc("/v1/kv/apps/haproxy/frontend/test/listenPort", handleFrontListenPort)
	mux.HandleFunc("/v1/kv/apps/haproxy/frontend/test/mode", handleFrontMode)
	mux.HandleFunc("/v1/kv/apps/haproxy/frontend/test/staticConf", handleFrontStaticConf)
	mux.HandleFunc("/v1/kv/apps/haproxy/backend/test/balance", handleBackBalance)
	mux.HandleFunc("/v1/kv/apps/haproxy/backend/test/catalogMapping", handleBackCatalogMapping)
	mux.HandleFunc("/v1/kv/apps/haproxy/backend/test/mode", handleFrontMode)
	mux.HandleFunc("/v1/kv/apps/haproxy/backend/test/staticConf", handleBackStaticConf)
	mux.HandleFunc("/v1/kv/apps/haproxy/backend/test/type", handleBackType)
	l, err := net.Listen("tcp", "127.0.0.1:12424")

	if err != nil {

	}
	testHTTPServer := httptest.Server{
		Listener: l,
		Config:   &http.Server{Handler: handlerAccessLog(mux)},
	}
	return &testHTTPServer
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
	body := `    option forwardfor
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
	body := `    option httpchk GET /systemHealth
    http-check expect string "success":true`
	w.Write([]byte(body))
}
func handleBackType(w http.ResponseWriter, r *http.Request) {
	writeHeaders(w, 200)
	body := `dynamic`
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
