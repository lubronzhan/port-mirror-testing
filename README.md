# port-mirror-testing

Idea from this https://gist.github.com/mcastelino/7d85f4164ffdaf48242f9281bb1d0f9b and https://geertj.blogspot.com/2010/12/network-security-monitoring-with-kvm.html. Example go code to mirror the traffic goes through one NIC to another NIC

# Prerequests
Golang 1.21+

# Test
```
# FROM_IF_NAME is the interface name where traffic will be mirrored from.
# TO_IF_NAME is the interface name where traffic will be mirrored to.
# FROM_IF_NAME_IP is the IP of interface where traffic will be mirrored from.

make build
sudo ./bin/port-mirror $FROM_IF_NAME $TO_IF_NAME
```

or

```
sudo su
go run main.go $FROM_IF_NAME $TO_IF_NAME
```

Then tcpdump
```
sudo tcpdump -nn -i ${TO_IF_NAME} -tttt port ${PORT} and host ${FROM_IF_NAME_IP}
```
