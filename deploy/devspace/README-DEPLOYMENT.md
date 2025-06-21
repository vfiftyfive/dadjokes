# DadJokes Deployment Guide

## 🚀 Durable Deployment Solution

This directory contains the durable deployment configuration that fixes common issues automatically.

### **Issues Fixed**

1. **Service Selector Mismatch**: Service now correctly selects pods with `app: joke-server`
2. **Region Variable Expansion**: `${REGION}` is properly substituted with actual region values
3. **NATS Timeout Protection**: Enhanced joke-worker with attempt limiting and fallback

### **Deployment Options**

#### **Option 1: Using DR Helper Functions (Recommended)**
```fish
# From the repository root
source infrastructure/scripts/k8s-dr-helpers.fish
dr-deploy milan
dr-deploy dublin
```

#### **Option 2: Direct DevSpace Deployment**
```fish
# From this directory
cd applications/dadjokes/deploy/devspace

# Set your region
export REGION=milan  # or dublin
export DEVSPACE_NAMESPACE=dev

# Deploy
devspace deploy
```

#### **Option 3: Using the Region-Specific Script**
```fish
# From this directory
./deploy-with-region.fish milan
./deploy-with-region.fish dublin
```

### **What Gets Deployed**

1. **MongoDB Operator & Instance**: Database with authentication
2. **Redis Operator & Instance**: Caching layer
3. **NATS**: Message broker for worker communication
4. **joke-server**: HTTP API server with proper region configuration
5. **joke-worker**: Background worker with NATS timeout protection
6. **Ingress**: ALB configuration for external access

### **Environment Variables**

| Variable | Description | Default |
|----------|-------------|---------|
| `REGION` | Deployment region | `milan` |
| `DEVSPACE_NAMESPACE` | K8s namespace | `dev` |
| `OPENAI_API_KEY` | OpenAI API key (encrypted) | From SOPS |

### **Verification**

After deployment, verify the system:

```fish
# Check all pods are running
kubectl get pods -n dev

# Test the endpoint
curl http://<ALB_URL>/joke

# Check logs for enhanced tracing
kubectl logs -n dev deployment/joke-worker --tail=20
```

### **Troubleshooting**

#### **503 Service Unavailable**
- **Cause**: Service selector mismatch
- **Fix**: Automatically handled by deployment scripts
- **Manual Fix**: `kubectl patch svc joke-server -n dev --type='merge' -p='{"spec":{"selector":{"app":"joke-server"}}}'`

#### **Region Shows ${REGION}**
- **Cause**: Variable not expanded in manifest
- **Fix**: Automatically handled by deployment scripts  
- **Manual Fix**: `kubectl set env deployment/joke-server -n dev REGION=<actual-region>`

#### **NATS Timeouts**
- **Cause**: Infinite loops in joke generation
- **Fix**: Enhanced joke-worker with attempt limiting deployed automatically
- **Monitoring**: Check logs for "Generation attempt X/3" messages

### **Files Structure**

```
deploy/devspace/
├── devspace.yaml              # Main DevSpace configuration
├── deploy-with-region.fish    # Region-specific deployment script
├── custom-resources/          # Kubernetes manifests
│   ├── joke-server-deployment.yaml  # Fixed with region placeholder
│   ├── joke-server-service.yaml     # Correct selector
│   └── ingress.yaml                 # ALB configuration
└── openai-api-key.enc.yaml   # Encrypted OpenAI key
```

### **Security**

- OpenAI API key is encrypted with SOPS
- Automatic decryption during deployment
- Keys stored as Kubernetes secrets

### **Multi-Region Deployment**

The system supports deployment to multiple regions simultaneously:

```fish
# Deploy to both regions
dr-deploy milan
dr-deploy dublin

# Check status of both
dr-status milan
dr-status dublin
```

Each region gets its own:
- EKS cluster
- Application deployment
- ALB endpoint
- Database instance 