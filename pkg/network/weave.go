package network

import (
  "bytes"
  "text/template"
)

type WeaveNetworkProvider struct {}

func NewWeaveNetworkProvider() (NetworkProvider) {
  return &WeaveNetworkProvider{}
}

func (fnp *WeaveNetworkProvider) Name() string {
  return "weave"
}

func (fnp *WeaveNetworkProvider) Create(podNetworkCidr string) (error) {

  // TODO: Ensure weave uses the API configured pod network else we have to rethink this interface...
  k8Definition, err := weave(podNetworkCidr)
  if err != nil {
    return err
  }
  return createk8objects(string(k8Definition[:]))
}

// Grab the resources for deploying a network
func weave(podNetworkCidr string) ([]byte, error) {

  data := struct {
    Network	string
  }{
    Network: podNetworkCidr,
  }
  // Provides an internet disconnected version...
  // From https://github.com/weaveworks/weave/releases/download/v1.9.4/weave-daemonset-k8s-1.6.yaml
  const weaveYaml = `kind: ClusterRole
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
          image: weaveworks/weave-kube:1.9.4
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
          image: weaveworks/weave-npc:1.9.4
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
      securityContext:
        seLinuxOptions:
          type: spc_t
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

  t := template.Must(template.New("weave").Parse(weaveYaml))
  var b bytes.Buffer
  if err := t.Execute(&b, data); err != nil {
    return b.Bytes(), err
  }

  return b.Bytes(), nil
}