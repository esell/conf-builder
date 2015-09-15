# Conf-builder
Builds out the HAproxy config file based on data in consul. Whenever a service change is detected a new config is created and HAproxy is reloaded.

# Configuration
Place the conf.json file in the same directory as the executable and run it, that's about it. The configurable values are: 

	haproxyReloadCmd = The command used to reload/restart haproxy
	vips = An array of strings that will be the VIPs you want in your HAproxy config. These should match the names used in the frontend/backend section of consul (see below)
	consulHostPort = The location of your consul server and the port if needed
	consulConfigPath = The root of your config in the consul key/value store. Everythign else hangs off of this (see below)
	configFile = the name/locatin of the HAproxy config file that will be output
	tempFile = the name/location of the temp file used during the config building process

# Consul layout

The expected consul layout would look like:

	/v1/kv/consulConfigPath/global = global section of the HAproxy config
	/v1/kv/consulConfigPath/defaults = defaults section of the HAProxy config
	/v1/kv/consulConfigPath/frontend/VIPname
										/bindOptions = any additional bind options to add (SSL, etc)
										/listenPort = port for HAProxy to listen on
										/mode = proxy type (tcp, http, etc)
										/staticConf = any static config you'd like to add
	/v1/kv/consulConfigPath/backend/VIPname
										/balance = balancer type (roundrobin, etc)
										/catalogMapping = consul service name
										/mode = proxy type (tcp, http, etc)
										/staticConf = any static config you'd like to add
										/type = dynamic/static member updates
                    
                    

