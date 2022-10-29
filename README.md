# singularity

A framework for running game servers on top of Kubernetes. This project is heavily inspired
by [Agones](https://github.com/googleforgames/agones). Singularity is licensed under [GNU AGPLv3](COPYING).

> **Note:** this project is heavily W.I.P and is not ready for production.

## Operator

Operator is the main component of **singularity**. It is responsible for reconciling CRDs within the cluster.

### CRDs

* **Fleet** manages multiple game servers (technically **GameServerSets**) by using the specified GameServer template.
  The fleet is responsible for rolling out updates. This can be compared to a **Deployment**.
* **GameServerSet** contains multiple **GameServers**. This can be compared with a **ReplicaSet**.
* **GameServer** manages a single game server (technically **Pod**).
  A GameServer may contain multiple **GameServerInstances**.
* **GameServerInstance** is owned by a **GameServer** and is the smallest "unit" within singularity.
  This can be used to host multiple games within the same Pod at once. 
