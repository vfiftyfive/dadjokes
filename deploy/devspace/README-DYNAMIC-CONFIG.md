# Dynamic MongoDB Configuration

## Overview
The MongoDB connection configuration is now dynamic and environment-driven using DevSpace variables.

## Files Created

### Environment Files (bash/zsh compatible)
- `.env.milan` - Milan-specific configuration (3 MongoDB members, primaryPreferred)
- `.env.dublin` - Dublin-specific configuration (2 MongoDB members, secondaryPreferred)  
- `.env.cross-region` - Cross-region configuration (5 members across both regions)

### Environment Files (fish shell compatible)
- `.env.milan.fish` - Milan configuration for fish shell
- `.env.dublin.fish` - Dublin configuration for fish shell

### ConfigMap
- `custom-resources/mongodb-connection.yaml` - Dynamic ConfigMap using DevSpace variables

## Usage

### For bash/zsh users:
```bash
# Deploy to Milan
cd applications/dadjokes/deploy/devspace
export $(cat .env.milan | xargs) && devspace deploy

# Deploy to Dublin  
export $(cat .env.dublin | xargs) && devspace deploy

# Deploy cross-region
export $(cat .env.cross-region | xargs) && devspace deploy
```

### For fish shell users:
```fish
# Deploy to Milan
cd applications/dadjokes/deploy/devspace
source .env.milan.fish && devspace deploy

# Deploy to Dublin
source .env.dublin.fish && devspace deploy
```

## Benefits
- No scripts needed
- Pure DevSpace functionality
- Environment-driven configuration
- Easy to switch between regions
- Flexible and extensible

## Variables Available
- `REGION` - milan/dublin
- `MONGO_HOSTS` - Comma-separated list of MongoDB hosts
- `MONGO_READ_PREFERENCE` - primaryPreferred/secondaryPreferred
- All standard MongoDB connection variables (username, password, etc.)

