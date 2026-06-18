# Learn Kubernetes Step by Step with OrbStack

A hands-on lab. Each step builds on the last. Run every command yourself, read the output, and break things on purpose — that's how this sticks.

Everything here runs on OrbStack's built-in single-node cluster on macOS. No cloud account, no cost.

---

## Step 0 — Setup

Install OrbStack (skip if you already have it):

```bash
brew install --cask orbstack
```

Open it once so it can finish setup (it installs `orb`, `docker`, and `kubectl` for you).

Turn on Kubernetes — it's off by default to save resources:

```bash
orb start k8s
```

Give it a minute, then confirm it's alive:

```bash
kubectl get nodes
```

You should see one node with status `Ready`. OrbStack already pointed your kubectl context at this cluster, so you don't need to configure anything.

```bash
kubectl config current-context     # should say "orbstack"
```

If you ever want to reset and start clean, you can stop/start k8s from the OrbStack app's Kubernetes settings.

---

## Step 1 — The mental model (read this once)

Kubernetes runs your app as a set of objects. The ones you'll actually touch early on:

- **Pod** — the smallest unit. One or more containers that run together. You rarely create these directly.
- **Deployment** — manages Pods for you: keeps N copies running, replaces crashed ones, handles rolling updates. This is what you usually create.
- **Service** — a stable network address + load balancer in front of a set of Pods. Pods come and go and change IP; a Service doesn't.
- **ConfigMap / Secret** — configuration and credentials injected into Pods, kept separate from the image.
- **Ingress** — routes outside HTTP traffic to Services by hostname/path.

The whole system is *declarative*: you describe the desired state in YAML, apply it, and Kubernetes works continuously to make reality match. Keep that idea in mind — most "magic" is just the controller reconciling toward what you declared.

Three commands you'll use constantly:

```bash
kubectl get <type>          # list things
kubectl describe <type> <name>   # detailed status + recent events
kubectl logs <pod>          # container output
```

---

## Step 2 — Your first Pod

Run a single nginx Pod imperatively:

```bash
kubectl run hello --image=nginx
kubectl get pods -w     # -w watches; Ctrl-C to stop once it's "Running"
```

Look at it in detail. The **Events** section at the bottom is your best friend when things go wrong:

```bash
kubectl describe pod hello
```

Get a shell *inside* the container:

```bash
kubectl exec -it hello -- bash
# inside: try `curl localhost`, then `exit`
```

Reach it from your Mac. OrbStack assigns Pods reachable addresses, but the simplest portable way is port-forward:

```bash
kubectl port-forward pod/hello 8080:80
# open http://localhost:8080 in your browser, then Ctrl-C
```

Now delete it:

```bash
kubectl delete pod hello
kubectl get pods    # gone, and it does NOT come back
```

That last point matters: a bare Pod is not self-healing. That's why we use Deployments.

---

## Step 3 — Deployments and self-healing

Time to write YAML. Create `deployment.yaml`:

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: web
spec:
  replicas: 3
  selector:
    matchLabels:
      app: web
  template:
    metadata:
      labels:
        app: web
    spec:
      containers:
        - name: nginx
          image: nginx:1.27
          ports:
            - containerPort: 80
```

Apply it and watch three Pods appear:

```bash
kubectl apply -f deployment.yaml
kubectl get pods -l app=web
```

**Now break it on purpose.** Delete one Pod and watch Kubernetes immediately replace it:

```bash
kubectl delete pod -l app=web --field-selector status.phase=Running | head -1
kubectl get pods -l app=web -w
```

You'll see a new Pod spin up to keep the count at 3. That's the reconciliation loop in action — you *declared* 3, so it maintains 3.

Scale it:

```bash
kubectl scale deployment web --replicas=5
kubectl get pods -l app=web
kubectl scale deployment web --replicas=2
```

Do a rolling update by changing the image version, then watch the rollout:

```bash
kubectl set image deployment/web nginx=nginx:1.28
kubectl rollout status deployment/web
kubectl rollout history deployment/web
```

And roll back if you don't like it:

```bash
kubectl rollout undo deployment/web
```

---

## Step 4 — Services (stable networking)

Your Pods have changing IPs. A Service gives them one stable front door.

Create `service.yaml`:

```yaml
apiVersion: v1
kind: Service
metadata:
  name: web
spec:
  type: LoadBalancer
  selector:
    app: web        # matches the Pod labels from Step 3
  ports:
    - port: 80
      targetPort: 80
```

```bash
kubectl apply -f service.yaml
kubectl get service web
```

Here's an OrbStack perk: on most local clusters a `LoadBalancer` Service stays stuck on `<pending>`. On OrbStack it gets a real reachable address out of the box. You can hit it directly from your Mac:

```bash
kubectl get service web   # note the EXTERNAL-IP
curl http://<EXTERNAL-IP>
```

OrbStack also gives you a zero-config hostname. LoadBalancer Services are reachable at `*.k8s.orb.local`:

```bash
curl http://web.default.k8s.orb.local
```

Notice the Service load-balances across all the Pods behind it — no extra work from you. Delete and recreate Pods (Step 3) and the Service keeps working because it selects by *label*, not by IP.

---

## Step 5 — ConfigMaps and Secrets

Keep configuration out of your image. Create `config.yaml`:

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: app-config
data:
  GREETING: "Hello from Kubernetes"
  LOG_LEVEL: "debug"
---
apiVersion: v1
kind: Secret
metadata:
  name: app-secret
type: Opaque
stringData:
  API_KEY: "super-secret-value"
```

```bash
kubectl apply -f config.yaml
kubectl get configmap app-config -o yaml
kubectl get secret app-secret -o yaml   # note the value is base64-encoded, not encrypted
```

Now inject them into a Pod as environment variables. Create `pod-with-config.yaml`:

```yaml
apiVersion: v1
kind: Pod
metadata:
  name: config-demo
spec:
  containers:
    - name: app
      image: busybox:1.36
      command: ["sh", "-c", "env | grep -E 'GREETING|LOG_LEVEL|API_KEY'; sleep 3600"]
      envFrom:
        - configMapRef:
            name: app-config
        - secretRef:
            name: app-secret
```

```bash
kubectl apply -f pod-with-config.yaml
kubectl logs config-demo     # you'll see your config + secret as env vars
```

The lesson: the same image can be reconfigured for dev/staging/prod just by swapping ConfigMaps and Secrets — nothing rebuilt.

---

## Step 6 — Ingress (HTTP routing by hostname)

A Service exposes one app. Ingress routes many apps behind one entry point by hostname or path.

OrbStack doesn't ship an Ingress controller by default, so install nginx-ingress:

```bash
kubectl apply -f https://raw.githubusercontent.com/kubernetes/ingress-nginx/main/deploy/static/provider/cloud/deploy.yaml
kubectl wait --namespace ingress-nginx \
  --for=condition=ready pod \
  --selector=app.kubernetes.io/component=controller \
  --timeout=120s
```

Create `ingress.yaml` (this needs a ClusterIP Service; change your Step 4 Service `type` to `ClusterIP` or add a second one — for simplicity, edit `web` to `type: ClusterIP` and re-apply):

```yaml
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: web-ingress
  annotations:
    nginx.ingress.kubernetes.io/rewrite-target: /
spec:
  ingressClassName: nginx
  rules:
    - host: myapp.k8s.orb.local
      http:
        paths:
          - path: /
            pathType: Prefix
            backend:
              service:
                name: web
                port:
                  number: 80
```

```bash
kubectl apply -f ingress.yaml
curl http://myapp.k8s.orb.local
```

OrbStack's `*.k8s.orb.local` wildcard points at LoadBalancers (including the ingress controller), so this resolves from your Mac with no `/etc/hosts` editing.

---

## Step 7 — A real two-tier app (putting it together)

Now combine everything: an app Deployment + Service, talking to a Redis Deployment + Service via in-cluster DNS.

Create `two-tier.yaml`:

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: redis
spec:
  replicas: 1
  selector:
    matchLabels: { app: redis }
  template:
    metadata:
      labels: { app: redis }
    spec:
      containers:
        - name: redis
          image: redis:7-alpine
          ports: [{ containerPort: 6379 }]
---
apiVersion: v1
kind: Service
metadata:
  name: redis
spec:
  selector: { app: redis }
  ports: [{ port: 6379, targetPort: 6379 }]
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: app
spec:
  replicas: 2
  selector:
    matchLabels: { app: app }
  template:
    metadata:
      labels: { app: app }
    spec:
      containers:
        - name: app
          image: redis:7-alpine     # reusing the image just to get redis-cli
          command: ["sleep", "3600"]
```

```bash
kubectl apply -f two-tier.yaml
kubectl get pods
```

Now prove that in-cluster DNS works. The app Pods can reach Redis just by its Service *name* — Kubernetes resolves `redis` to the Service IP:

```bash
APP_POD=$(kubectl get pod -l app=app -o jsonpath='{.items[0].metadata.name}')
kubectl exec -it $APP_POD -- redis-cli -h redis ping     # -> PONG
kubectl exec -it $APP_POD -- redis-cli -h redis set msg "it works"
kubectl exec -it $APP_POD -- redis-cli -h redis get msg  # -> "it works"
```

The full DNS name is `redis.default.svc.cluster.local`, and OrbStack makes those `cluster.local` names reachable from your Mac too. The short name `redis` works inside the cluster because Pods default to searching their own namespace.

---

## Step 8 — Your debugging toolkit

When something's broken, this is the loop. Memorize it:

```bash
kubectl get pods                       # what state is it in?
kubectl describe pod <name>            # read the Events at the bottom
kubectl logs <name>                    # what did the app say?
kubectl logs <name> --previous         # logs from a crashed/restarted container
kubectl get events --sort-by=.lastTimestamp   # cluster-wide recent events
```

Common states and what they mean:
- `ImagePullBackOff` / `ErrImagePull` — wrong image name/tag, or it can't be pulled.
- `CrashLoopBackOff` — the container starts then dies repeatedly. Check `logs --previous`.
- `Pending` — can't be scheduled (usually resources, on a single node rarely an issue).
- `ContainerCreating` stuck — often a missing ConfigMap/Secret it's mounting.

**OrbStack-specific gotcha worth remembering:** images tagged `:latest` are always re-pulled. When you build an image locally and want the cluster to use it without pushing to a registry, give it a real tag (`:dev`, `:1`) or set `imagePullPolicy: IfNotPresent`. Because OrbStack shares its container engine with the cluster, locally built images are available to Pods immediately — no registry push needed.

---

## Step 9 — Cleanup

Delete everything you created in this lab:

```bash
kubectl delete -f two-tier.yaml
kubectl delete -f ingress.yaml
kubectl delete -f config.yaml
kubectl delete -f service.yaml
kubectl delete -f deployment.yaml
kubectl delete pod config-demo
```

Or nuke the ingress controller too:

```bash
kubectl delete -f https://raw.githubusercontent.com/kubernetes/ingress-nginx/main/deploy/static/provider/cloud/deploy.yaml
```

If you want a totally fresh cluster, stop and restart Kubernetes from OrbStack's settings.

---

## Where to go next

You've now touched every core object. To go deeper:

1. **Namespaces & resource limits** — isolate workloads, set CPU/memory requests and limits.
2. **Probes** — add `livenessProbe` and `readinessProbe` so Kubernetes knows when a Pod is healthy.
3. **StatefulSets & PersistentVolumes** — for databases and anything that needs stable storage/identity.
4. **Helm** — package and template your manifests (`brew install helm`).
5. **Multi-node clusters** — OrbStack's built-in cluster is single-node, so things like node affinity, taints/tolerations, and pod anti-affinity can't really be demonstrated. Run `kind` or `k3d` inside an OrbStack Linux machine to get multiple nodes when you're ready.

A good habit: any time you run an imperative `kubectl` command, ask "what's the YAML equivalent?" Real clusters are managed declaratively through files in version control, not one-off commands.

Work through the steps in order, and don't skip the "break it on purpose" parts — watching Kubernetes recover is where the declarative model finally clicks.
