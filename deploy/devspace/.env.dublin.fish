set -x REGION dublin
set -x MONGO_HOSTS mongodb-0.mongodb-svc.dev.svc.cluster.local:27017,mongodb-1.mongodb-svc.dev.svc.cluster.local:27017,mongodb-0.milan.mongodb.internal.k8sdr.com:27017,mongodb-1.milan.mongodb.internal.k8sdr.com:27017,mongodb-2.milan.mongodb.internal.k8sdr.com:27017
set -x MONGO_READ_PREFERENCE secondaryPreferred
