set -x REGION dublin
set -x MONGO_HOSTS mongodb-0.mongodb-svc.dev.svc.cluster.local:27017,mongodb-1.mongodb-svc.dev.svc.cluster.local:27017,mongodb-milan.internal.k8sdr.com:27017
set -x MONGO_READ_PREFERENCE primaryPreferred
set -x MONGODB_MEMBERS 2
