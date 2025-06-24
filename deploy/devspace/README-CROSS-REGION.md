# Cross-Region MongoDB Configuration

## Overview
Each region connects to all 5 MongoDB replicas using optimized DNS:
- **Local members**: Fast cluster DNS (`svc.cluster.local`)  
- **Remote members**: Private hosted zone DNS (`*.mongodb.internal.k8sdr.com`)

## Environment Files

### Milan Perspective (5 replicas)
**`.env.milan`** - Milan apps connecting to 5-member cross-region RS:
- 3 local Milan members via `mongodb-svc.dev.svc.cluster.local`
- 2 remote Dublin members via `dublin.mongodb.internal.k8sdr.com`
- Read preference: `primaryPreferred` (Milan is primary region)

### Dublin Perspective (5 replicas)  
**`.env.dublin`** - Dublin apps connecting to 5-member cross-region RS:
- 2 local Dublin members via `mongodb-svc.dev.svc.cluster.local`
- 3 remote Milan members via `milan.mongodb.internal.k8sdr.com`  
- Read preference: `secondaryPreferred` (Dublin is secondary region)

## Usage

### Fish Shell
```fish
# Deploy to Milan with cross-region awareness
kubectl config use-context milan
cd applications/dadjokes/deploy/devspace
source .env.milan.fish && devspace deploy

# Deploy to Dublin with cross-region awareness
kubectl config use-context dublin
source .env.dublin.fish && devspace deploy
```

### Bash/Zsh
```bash
# Deploy to Milan
export $(cat .env.milan | xargs) && devspace deploy

# Deploy to Dublin  
export $(cat .env.dublin | xargs) && devspace deploy
```

## Benefits
- **Optimized latency**: Local members use fast cluster DNS
- **Cross-region failover**: Remote members accessible via private zone
- **Region-aware**: Each region prefers its local members
- **True DR**: Apps in both regions can survive complete region failure

## Connection Strings Generated
**Milan apps see:**
```
mongodb://demo:spectrocloud@mongodb-0.mongodb-svc.dev.svc.cluster.local:27017,mongodb-1.mongodb-svc.dev.svc.cluster.local:27017,mongodb-2.mongodb-svc.dev.svc.cluster.local:27017,mongodb-0.dublin.mongodb.internal.k8sdr.com:27017,mongodb-1.dublin.mongodb.internal.k8sdr.com:27017/jokesdb?readPreference=primaryPreferred&replicaSet=mongodb
```

**Dublin apps see:**
```
mongodb://demo:spectrocloud@mongodb-0.mongodb-svc.dev.svc.cluster.local:27017,mongodb-1.mongodb-svc.dev.svc.cluster.local:27017,mongodb-0.milan.mongodb.internal.k8sdr.com:27017,mongodb-1.milan.mongodb.internal.k8sdr.com:27017,mongodb-2.milan.mongodb.internal.k8sdr.com:27017/jokesdb?readPreference=secondaryPreferred&replicaSet=mongodb
```

