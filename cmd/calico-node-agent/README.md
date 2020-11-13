## Run test

```
# generate config
$ cat <<EOF > ./config.yaml
nodeName: $(hostname)
ipPools:
- cidr: 192.168.122.34/32
- cidr: 192.168.122.35/32
EOF

# set calico connection environment
$ export DATASTORE_TYPE=kubernetes
$ export KUBECONFIG=~/.kube/config

# start calico-node-agent
$ ./_output/bin/calico-node-agent -conf ./config.yaml

# check calico ipPools
$ calicoctl get ippools  -o wide
NAME                          CIDR                NAT     IPIPMODE   VXLANMODE   DISABLED   SELECTOR
default-ipv4-ippool           10.40.0.0/16        true    Always     Never       false      all()
lzx-t470p-192-168-122-34-32   192.168.122.34/32   false   Never      Never       false      kubernetes.io/hostname == "lzx-t470p"
lzx-t470p-192-168-122-35-32   192.168.122.35/32   false   Never      Never       false      kubernetes.io/hostname == "lzx-t470p"
```
