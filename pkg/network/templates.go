package network

const flannelYaml = `---
# Create the clusterrole and clusterrolebinding:
# $ kubectl create -f kube-flannel-rbac.yml
# Create the pod using the same namespace used by the flannel serviceaccount:
# $ kubectl create --namespace kube-system -f kube-flannel.yml
---
kind: ClusterRole
apiVersion: rbac.authorization.k8s.io/v1beta1
metadata:
  name: flannel
rules:
  - apiGroups:
      - ""
    resources:
      - pods
    verbs:
      - get
  - apiGroups:
      - ""
    resources:
      - nodes
    verbs:
      - list
      - watch
  - apiGroups:
      - ""
    resources:
      - nodes/status
    verbs:
      - patch
---
kind: ClusterRoleBinding
apiVersion: rbac.authorization.k8s.io/v1beta1
metadata:
  name: flannel
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: ClusterRole
  name: flannel
subjects:
- kind: ServiceAccount
  name: flannel
  namespace: kube-system
---
apiVersion: v1
kind: ServiceAccount
metadata:
  name: flannel
  namespace: kube-system
---
kind: ConfigMap
apiVersion: v1
metadata:
  name: kube-flannel-cfg
  namespace: kube-system
  labels:
    tier: node
    app: flannel
data:
  cni-conf.json: |
    {
      "name": "cbr0",
      "type": "flannel",
      "delegate": {
        "isDefaultGateway": true
      }
    }
  net-conf.json: |
    {
      "Network": "{{ .Network }}",
      "Backend": {
        "Type": "vxlan"
      }
    }
---
apiVersion: extensions/v1beta1
kind: DaemonSet
metadata:
  name: kube-flannel-ds
  namespace: kube-system
  labels:
    tier: node
    app: flannel
spec:
  template:
    metadata:
      labels:
        tier: node
        app: flannel
    spec:
      hostNetwork: true
      nodeSelector:
        beta.kubernetes.io/arch: amd64
      tolerations:
      - key: node-role.kubernetes.io/master
        operator: Exists
        effect: NoSchedule
      serviceAccountName: flannel
      containers:
      - name: kube-flannel
        image: quay.io/coreos/flannel:v0.7.1-amd64
        command: [ "/opt/bin/flanneld", "--ip-masq", "--kube-subnet-mgr" ]
        securityContext:
          privileged: true
        env:
        - name: POD_NAME
          valueFrom:
            fieldRef:
              fieldPath: metadata.name
        - name: POD_NAMESPACE
          valueFrom:
            fieldRef:
              fieldPath: metadata.namespace
        volumeMounts:
        - name: run
          mountPath: /run
        - name: flannel-cfg
          mountPath: /etc/kube-flannel/
      - name: install-cni
        image: quay.io/coreos/flannel:v0.7.1-amd64
        command: [ "/bin/sh", "-c", "set -e -x; cp -f /etc/kube-flannel/cni-conf.json /etc/cni/net.d/10-flannel.conf; while true; do sleep 3600; done" ]
        volumeMounts:
        - name: cni
          mountPath: /etc/cni/net.d
        - name: flannel-cfg
          mountPath: /etc/kube-flannel/
      volumes:
        - name: run
          hostPath:
            path: /run
        - name: cni
          hostPath:
            path: /etc/cni/net.d
        - name: flannel-cfg
          configMap:
            name: kube-flannel-cfg`

const canalYaml = `# Calico Roles
kind: ClusterRole
apiVersion: rbac.authorization.k8s.io/v1beta1
metadata:
  name: canal
  namespace: kube-system
rules:
  - apiGroups: [""]
    resources:
      - namespaces
    verbs:
      - get
      - list
      - watch
  - apiGroups: [""]
    resources:
      - pods/status
    verbs:
      - update
  - apiGroups: [""]
    resources:
      - pods
    verbs:
      - get
      - list
      - watch
  - apiGroups: [""]
    resources:
      - nodes
    verbs:
      - get
      - list
      - update
      - watch
  - apiGroups: ["extensions"]
    resources:
      - thirdpartyresources
    verbs:
      - create
      - get
      - list
      - watch
  - apiGroups: ["extensions"]
    resources:
      - networkpolicies
    verbs:
      - get
      - list
      - watch
  - apiGroups: ["projectcalico.org"]
    resources:
      - globalconfigs
    verbs:
      - create
      - get
      - list
      - update
      - watch
  - apiGroups: ["projectcalico.org"]
    resources:
      - ippools
    verbs:
      - create
      - delete
      - get
      - list
      - update
      - watch

---

# Flannel roles
# Pulled from https://github.com/coreos/flannel/blob/master/Documentation/kube-flannel-rbac.yml
kind: ClusterRole
apiVersion: rbac.authorization.k8s.io/v1beta1
metadata:
  name: flannel
rules:
  - apiGroups:
      - ""
    resources:
      - pods
    verbs:
      - get
  - apiGroups:
      - ""
    resources:
      - nodes
    verbs:
      - list
      - watch
  - apiGroups:
      - ""
    resources:
      - nodes/status
    verbs:
      - patch
---

kind: ClusterRoleBinding
apiVersion: rbac.authorization.k8s.io/v1beta1
metadata:
  name: flannel
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: ClusterRole
  name: flannel
subjects:
- kind: ServiceAccount
  name: canal
  namespace: kube-system

---

apiVersion: rbac.authorization.k8s.io/v1beta1
kind: ClusterRoleBinding
metadata:
  name: canal
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: ClusterRole
  name: canal
subjects:
- kind: ServiceAccount
  name: canal
  namespace: kube-system

---
# This ConfigMap can be used to configure a self-hosted Canal installation.
kind: ConfigMap
apiVersion: v1
metadata:
  name: canal-config
  namespace: kube-system
data:
  # The interface used by canal for host <-> host communication.
  # If left blank, then the interface is chosen using the node's
  # default route.
  canal_iface: ""

  # Whether or not to masquerade traffic to destinations not within
  # the pod network.
  masquerade: "true"

  # The CNI network configuration to install on each node.
  cni_network_config: |-
    {
        "name": "k8s-pod-network",
        "type": "calico",
        "log_level": "info",
        "datastore_type": "kubernetes",
        "hostname": "__KUBERNETES_NODE_NAME__",
        "ipam": {
            "type": "host-local",
            "subnet": "usePodCidr"
        },
        "policy": {
            "type": "k8s",
            "k8s_auth_token": "__SERVICEACCOUNT_TOKEN__"
        },
        "kubernetes": {
            "k8s_api_root": "https://__KUBERNETES_SERVICE_HOST__:__KUBERNETES_SERVICE_PORT__",
            "kubeconfig": "__KUBECONFIG_FILEPATH__"
        }
    }

  # Flannel network configuration. Mounted into the flannel container.
  net-conf.json: |
    {
      "Network": "{{ .Network }}",
      "Backend": {
        "Type": "vxlan"
      }
    }

---

# This manifest installs the calico/node container, as well
# as the Calico CNI plugins and network config on
# each master and worker node in a Kubernetes cluster.
kind: DaemonSet
apiVersion: extensions/v1beta1
metadata:
  name: canal
  namespace: kube-system
  labels:
    k8s-app: canal
spec:
  selector:
    matchLabels:
      k8s-app: canal
  template:
    metadata:
      labels:
        k8s-app: canal
      annotations:
        scheduler.alpha.kubernetes.io/critical-pod: ''
    spec:
      hostNetwork: true
      serviceAccountName: canal
      tolerations:
        # Allow the pod to run on the master.  This is required for
        # the master to communicate with pods.
        - key: node-role.kubernetes.io/master
          effect: NoSchedule
        # Mark the pod as a critical add-on for rescheduling.
        - key: "CriticalAddonsOnly"
          operator: "Exists"
      containers:
        # Runs calico/node container on each Kubernetes node.  This
        # container programs network policy and routes on each
        # host.
        - name: calico-node
          image: quay.io/calico/node:v1.2.1
          env:
            # Use Kubernetes API as the backing datastore.
            - name: DATASTORE_TYPE
              value: "kubernetes"
            # Enable felix logging.
            - name: FELIX_LOGSEVERITYSYS
              value: "info"
            # Period, in seconds, at which felix re-applies all iptables state
            - name: FELIX_IPTABLESREFRESHINTERVAL
              value: "60"
            # Disable IPV6 support in Felix.
            - name: FELIX_IPV6SUPPORT
              value: "false"
            # Don't enable BGP.
            - name: CALICO_NETWORKING_BACKEND
              value: "none"
            # Disable file logging so 'kubectl logs' works.
            - name: CALICO_DISABLE_FILE_LOGGING
              value: "true"
            - name: WAIT_FOR_DATASTORE
              value: "true"
            # No IP address needed.
            - name: IP
              value: ""
            - name: HOSTNAME
              valueFrom:
                fieldRef:
                  fieldPath: spec.nodeName
            # Set Felix endpoint to host default action to ACCEPT.
            - name: FELIX_DEFAULTENDPOINTTOHOSTACTION
              value: "ACCEPT"
          securityContext:
            privileged: true
          resources:
            requests:
              cpu: 250m
          volumeMounts:
            - mountPath: /lib/modules
              name: lib-modules
              readOnly: true
            - mountPath: /var/run/calico
              name: var-run-calico
              readOnly: false
        # This allows setting strict filtering for calico for all interfaces
        - name: strict-rp-filter
          image: busybox
          command: ["sh", "-c", "echo 1 >/host/proc/sys/net/ipv4/conf/all/rp_filter; while true ; do sleep 100 ; done"]
          securityContext:
            privileged: true
          volumeMounts:
            - mountPath: /host/proc
              name: proc
        # This container installs the Calico CNI binaries
        # and CNI network config file on each node.
        - name: install-cni
          image: quay.io/calico/cni:v1.8.3
          command: ["/install-cni.sh"]
          env:
            # The CNI network config to install on each node.
            - name: CNI_NETWORK_CONFIG
              valueFrom:
                configMapKeyRef:
                  name: canal-config
                  key: cni_network_config
            - name: KUBERNETES_NODE_NAME
              valueFrom:
                fieldRef:
                  fieldPath: spec.nodeName
          volumeMounts:
            - mountPath: /host/opt/cni/bin
              name: cni-bin-dir
            - mountPath: /host/etc/cni/net.d
              name: cni-net-dir
        # This container runs flannel using the kube-subnet-mgr backend
        # for allocating subnets.
        - name: kube-flannel
          image: quay.io/coreos/flannel:v0.8.0
          command: [ "/opt/bin/flanneld", "--ip-masq", "--kube-subnet-mgr" ]
          securityContext:
            privileged: true
          env:
            - name: POD_NAME
              valueFrom:
                fieldRef:
                  fieldPath: metadata.name
            - name: POD_NAMESPACE
              valueFrom:
                fieldRef:
                  fieldPath: metadata.namespace
            - name: FLANNELD_IFACE
              valueFrom:
                configMapKeyRef:
                  name: canal-config
                  key: canal_iface
            - name: FLANNELD_IP_MASQ
              valueFrom:
                configMapKeyRef:
                  name: canal-config
                  key: masquerade
          volumeMounts:
          - name: run
            mountPath: /run
          - name: flannel-cfg
            mountPath: /etc/kube-flannel/
      volumes:
        # Used by calico/node.
        - name: lib-modules
          hostPath:
            path: /lib/modules
        - name: var-run-calico
          hostPath:
            path: /var/run/calico
        # Used to install CNI.
        - name: cni-bin-dir
          hostPath:
            path: /opt/cni/bin
        - name: cni-net-dir
          hostPath:
            path: /etc/cni/net.d
        # Used by flannel.
        - name: run
          hostPath:
            path: /run
        - name: flannel-cfg
          configMap:
            name: canal-config
        - name: proc
          hostPath:
            path: /proc


---

apiVersion: v1
kind: ServiceAccount
metadata:
  name: canal
  namespace: kube-system
`

const weaveYaml = `# From https://github.com/weaveworks/weave/releases/download/v1.9.5/weave-daemonset-k8s-1.6.yaml
kind: ClusterRole
apiVersion: rbac.authorization.k8s.io/v1beta1
metadata:
  name: weave-net
rules:
- apiGroups:
  - ""
  resources:
  - pods
  - namespaces
  - nodes
  verbs:
  - get
  - list
  - watch
- apiGroups:
  - extensions
  resources:
  - networkpolicies
  verbs:
  - get
  - list
  - watch
---
apiVersion: v1
kind: ServiceAccount
metadata:
  name: weave-net
  namespace: kube-system
---
kind: ClusterRoleBinding
apiVersion: rbac.authorization.k8s.io/v1beta1
metadata:
  name: weave-net
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: ClusterRole
  name: weave-net
subjects:
- kind: ServiceAccount
  name: weave-net
  namespace: kube-system
---
apiVersion: extensions/v1beta1
kind: DaemonSet
metadata:
  name: weave-net
  namespace: kube-system
spec:
  template:
    metadata:
      labels:
        name: weave-net
    spec:
      hostNetwork: true
      hostPID: true
      containers:
        - name: weave
          image: weaveworks/weave-kube:1.9.5
          command:
            - /home/weave/launch.sh
          livenessProbe:
            initialDelaySeconds: 30
            httpGet:
              host: 127.0.0.1
              path: /status
              port: 6784
          securityContext:
            privileged: true
          volumeMounts:
            - name: weavedb
              mountPath: /weavedb
            - name: cni-bin
              mountPath: /host/opt
            - name: cni-bin2
              mountPath: /host/home
            - name: cni-conf
              mountPath: /host/etc
            - name: dbus
              mountPath: /host/var/lib/dbus
            - name: lib-modules
              mountPath: /lib/modules
          resources:
            requests:
              cpu: 10m
        - name: weave-npc
          image: weaveworks/weave-npc:1.9.5
          resources:
            requests:
              cpu: 10m
          securityContext:
            privileged: true
      restartPolicy: Always
      tolerations:
      - key: node-role.kubernetes.io/master
        effect: NoSchedule
      serviceAccountName: weave-net
      # TODO: Investigate problem with CoreOS / Kubelet-wrapper, see https://github.com/coreos/bugs/issues/1580
      #securityContext:
      #  seLinuxOptions:
      #    type: spc_t
      volumes:
        - name: weavedb
          emptyDir: {}
        - name: cni-bin
          hostPath:
            path: /opt
        - name: cni-bin2
          hostPath:
            path: /home
        - name: cni-conf
          hostPath:
            path: /etc
        - name: dbus
          hostPath:
            path: /var/lib/dbus
        - name: lib-modules
          hostPath:
            path: /lib/modules
`
