set -x REGION milan
set -x MONGO_HOSTS mongodb-0.mongodb-svc.dev.svc.cluster.local:27017,mongodb-1.mongodb-svc.dev.svc.cluster.local:27017,mongodb-2.mongodb-svc.dev.svc.cluster.local:27017,mongodb-0.dublin.mongodb.internal.k8sdr.com:27017,mongodb-1.dublin.mongodb.internal.k8sdr.com:27017
set -x MONGO_READ_PREFERENCE primaryPreferred
