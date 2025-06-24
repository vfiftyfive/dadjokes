# Cross-Region Deployment Guide

## Prerequisites

1. **Ensure both clusters are running**:
```bash
kubectl config get-contexts | grep -E "milan|dublin"
```

2. **Verify external-dns is running in both regions**:
```bash
kubectl get pods -n kube-system -l app.kubernetes.io/name=external-dns --context=milan
kubectl get pods -n kube-system -l app.kubernetes.io/name=external-dns --context=dublin
```

## Deployment Order of Operations

### Step 1: Deploy Milan (Primary Region) First

```bash
cd /Users/nvermande/Documents/Dev/k8spocalypse/applications/dadjokes/deploy/devspace

# Set Milan environment
source .env.milan.fish

# Switch to Milan context
kubectl config use-context milan

# Create namespace if needed
kubectl create namespace dev --dry-run=client -o yaml | kubectl apply -f -

# Deploy to Milan
devspace deploy
```

This will:
1. Deploy MongoDB operator
2. Deploy Redis operator
3. Create MongoDB with 3 members
4. Create Redis instance
5. Deploy NATS
6. Deploy joke-server and joke-worker
7. Annotate MongoDB service for external-dns: `mongodb.milan.mongodb.internal.k8sdr.com`

### Step 2: Verify Milan MongoDB is Ready

```bash
# Check MongoDB pods
kubectl get pods -n dev | grep mongodb

# Check replica set status
kubectl exec -it mongodb-0 -n dev -- mongosh \
  "mongodb://demo:spectrocloud@localhost:27017/admin?authSource=admin" \
  --eval "rs.status()"

# Verify DNS records were created
aws route53 list-resource-record-sets --hosted-zone-id Z10469153IJ9YY6JW853I \
  --query "ResourceRecordSets[?contains(Name, 'milan.mongodb')]" --output table
```

### Step 3: Deploy Dublin (DR Region)

```bash
# Set Dublin environment
source .env.dublin.fish

# Switch to Dublin context
kubectl config use-context dublin

# Create namespace
kubectl create namespace dev --dry-run=client -o yaml | kubectl apply -f -

# Deploy to Dublin
devspace deploy
```

This will:
1. Deploy MongoDB operator
2. Deploy Redis operator
3. Create MongoDB with 2 members (using mongodb-dublin.yaml)
4. Create Redis instance
5. Deploy NATS
6. Deploy joke-server and joke-worker
7. Annotate MongoDB service for external-dns: `mongodb.dublin.mongodb.internal.k8sdr.com`
8. **Run the mongodb-join-milan Job** (only in Dublin)

### Step 4: Monitor the Join Process

```bash
# Watch the join job
kubectl logs -f job/mongodb-dublin-join-milan -n dev

# Check if operator was scaled down
kubectl get deployment mongodb-kubernetes-operator -n dev

# Verify Dublin pods joined Milan replica set
kubectl exec -it mongodb-0 -n dev --context=milan -- mongosh \
  "mongodb://demo:spectrocloud@localhost:27017/admin?authSource=admin" \
  --eval "rs.status()"
```

### Step 5: Verify Cross-Region Setup

```bash
# Check all DNS records
aws route53 list-resource-record-sets --hosted-zone-id Z10469153IJ9YY6JW853I \
  --query "ResourceRecordSets[?contains(Name, 'mongodb') && Type == 'A']" --output table

# Test connectivity from Dublin to Milan
kubectl run -it --rm debug --image=mongo:6.0.5 --restart=Never --context=dublin -- \
  mongosh "mongodb://demo:spectrocloud@mongodb-0.milan.mongodb.internal.k8sdr.com:27017/admin?authSource=admin" \
  --eval "db.adminCommand('ping')"

# Check replication status
kubectl exec -it mongodb-0 -n dev --context=milan -- mongosh \
  "mongodb://demo:spectrocloud@localhost:27017/admin?authSource=admin" \
  --eval "rs.printSecondaryReplicationInfo()"
```

### Step 6: Test the Application

```bash
# Get ingress endpoints
kubectl get ingress -n dev --context=milan
kubectl get ingress -n dev --context=dublin

# Test the application in both regions
curl http://<milan-ingress>/jokes
curl http://<dublin-ingress>/jokes
```

## Expected Final State

1. **MongoDB Replica Set**:
   - 5 total members
   - Milan: 3 voting members (PRIMARY + 2 SECONDARY)
   - Dublin: 2 non-voting members (SECONDARY with priority=0, votes=0)

2. **DNS Records**:
   - `mongodb-milan.internal.k8sdr.com` → Milan NLB
   - `mongodb-dublin.internal.k8sdr.com` → Dublin NLB
   - `mongodb.milan.mongodb.internal.k8sdr.com` → All Milan pod IPs
   - `mongodb-0.mongodb.milan.mongodb.internal.k8sdr.com` → Specific pod IP
   - (Similar pattern for Dublin)

3. **Application Behavior**:
   - Milan apps: Write to primary, read from any Milan member
   - Dublin apps: Cannot write (secondaries), read from local Dublin members

## Troubleshooting

If the join job fails:
```bash
# Check job logs
kubectl logs job/mongodb-dublin-join-milan -n dev --context=dublin

# Manually scale down operator if needed
kubectl scale deployment mongodb-kubernetes-operator -n dev --replicas=0 --context=dublin

# Retry the join manually
kubectl exec -it mongodb-dublin-0 -n dev --context=dublin -- mongosh \
  "mongodb://demo:spectrocloud@mongodb-0.milan.mongodb.internal.k8sdr.com:27017/admin?authSource=admin"
```

## Clean Up (if needed)

```bash
# Clean up Milan
kubectl config use-context milan
devspace purge

# Clean up Dublin  
kubectl config use-context dublin
devspace purge
```

The key is deploying Milan first and letting it stabilize before deploying Dublin. The Job in Dublin will automatically handle joining the Milan replica set!