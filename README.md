# Conf-builder
Builds out the HAproxy config file based on the data in consul. Whenever a service change is detected a new config is created and HAproxy is reloaded.

The expected consul layout would look like:

	/v1/kv/apps/haproxy/global = global section of the HAproxy config
	/v1/kv/apps/haproxy/defaults = defaults seciont of the HAProxy config
	/v1/kv/apps/haproxy/frontend/VIPname
										/bindOptions = any additional bind options to add (SSL, etc)
										/listenPort = port for HAProxy to listen on
										/mode = proxy type (tcp, http, etc)
										/staticConf = any static config you'd like to add
	/v1/kv/apps/haproxy/backend/VIPname
										/balance = balancer type (roundrobin, etc)
										/catalogMapping = consul service name
										/mode = proxy type (tcp, http, etc)
										/staticConf = any static config you'd like to add
										/type = dynamic/static member updates
                    
                    

