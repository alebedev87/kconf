# kconf

### Build

```bash
$ go build -o kconf main.go
$ sudo mv kconf /usr/local/bin/
```

### Add

```bash
$ kconf /home/bob/git/deployment/env/bob/kube_config_cluster.yml my
$ kconf
  1) my
$ ls -l /home/bob/.kconf/my
lrwxrwxrwx 1 bob bob 95 Apr 01 21:00 /home/bob/.kconf/my -> /home/bob/git/deployment/env/bob/kube_config_cluster.yml
```

## Set

```bash
$ kconf
  1) monit
  2) my
  3) my-down
  4) prod
$ `kconf 2`
$ echo $KUBECONFIG
/home/bob/.kconf/my
$ kconf
  1) monit
* 2) my
  3) my-down
  4) prod
$ kubectl get pods
```
