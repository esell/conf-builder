#!/bin/bash

# use the suggested method from HAProxy to properly kill traffic
iptables -I INPUT -p tcp -i eth1 --syn -j DROP
sleep 0.5
/usr/sbin/haproxy -f /etc/haproxy/haproxy.cfg -p /var/run/haproxy.pid -sf $(cat /var/run/haproxy.pid)
iptables -D INPUT -p tcp -i eth1 --syn -j DROP
