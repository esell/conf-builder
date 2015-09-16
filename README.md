# Conf-builder
Builds out the HAproxy config file based on data in consul. Whenever a service change is detected a new config is created and HAproxy is reloaded.  

## Building  
Conf-builder currently uses [gb](http://getgb.io/) to build although no outside libraries are needed:  

	go get github.com/constabulary/gb/...
	git clone https://github.com/radiantiq/conf-builder.git  
	cd conf-builder  
	gb test
	gb build

## Configuration
Place the conf.json file in the same directory as the executable and run it, that's about it. The configurable values are: 

`haproxyReloadCmd`
* The command used to reload/restart haproxy  

`vips`
* An array of strings that will be the VIPs you want in your HAproxy config. These should match the names used in the frontend/backend section of consul (see below)  

`consulHostPort`
* The location of your consul server and the port if needed  

`consulConfigPath`
* The root of your config in the consul key/value store. Everythign else hangs off of this (see below)  

`configFile`
* The name/locatin of the HAproxy config file that will be output  

`tempFile`
* The name/location of the temp file used during the config building process  

## Consul layout

The expected consul layout would look like:

	/v1/kv/consulConfigPath
	├── backend
	│   └── myApp
	│       ├── balance = balancer type (roundrobin, etc)
	│       ├── catalogMapping = consul service name
	│       ├── mode = proxy type (tcp, http, etc)
	│       ├── staticConf = any static config you'd like to add
	│       └── type = dynamic/static member updates
	├── defaults = defaults section of the HAProxy config
	├── frontend
	│   └── myApp
	│       ├── bindOptions = any additional bind options to add (SSL, etc)
	│       ├── listenPort port for HAProxy to listen on
	│       ├── mode = proxy type (tcp, http, etc)
	│       └── staticConf = any static config you'd like to add
	└── global = global section of the HAproxy config

Where `myApp` is the name you want to use for your VIP. You do not have to have a frontend AND a backend, you can just use one or the other if you'd like and of course you can have multiples (`myApp`, `anotherApp`, `yetAnother`, etc) as long as they follow the layout.
