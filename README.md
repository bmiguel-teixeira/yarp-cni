# yarp-cni - Yet Another Random Plugin (Kubernetes CNI Edition)

A random, simplied, and most likely kinda broken CNI implementation I started as a review work for CKA exam.
TLDR: It works, its just not finished.


# Components

The same binary serves 2 main functions via the `PLUGIN_MODE` environment variable.

## `PLUGIN_MODE` == `ROUTER` (WIP)

This mode is intended to run as daemon-set. In this mode, each pod (on each Node) will do the following actions:

* On startup, it checks the ConfigMap `yarp-routing-table` and allocates a subnet for the node it lives on (for example a /24) and registers it back on the ConfigMap. The structure should be something like this:
```
apiVersion: v1
kind: ConfigMap
metadata:
  name: yarp-routing-table
  namespace: kube-system
data:
  master1: 10.244.0.0/24
  node1: 10.244.1.0/24
  node2: 10.244.2.0/24
```
* On startup, it writes an `IPAM` db file locally to the node with metadata information to be leverage by the CNI mode. The DB structure is:
```
{
  "CIDR": "10.244.1.0/24",
  "AllocatedIps": []
}
```

* A long-living process that monitors for changes in the `yarp-routing-table` ConfigMap and watches for changes in Nodes. On changes, it will ensure that the local ip-routes are up-to-date to ensure all pods can reach each other over the network. It essentially implements a Router/RoutingTable.


## `PLUGIN_MODE` == `CNI`

This mode is invoked by Kubelet during Pod setup. This project currently implements spec `0.3.1`. For more information check https://github.com/containernetworking/cni/blob/master/SPEC.md

IPAM is managed via the `ipam.db` file. Its similar to the `host-local`.

