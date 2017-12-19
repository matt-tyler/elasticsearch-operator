# Example Run #

```bash
# Create a cluster in the target account
env PROJECT=<PROJECT> ZONE=<ZONE> ./e2e --up

# Get credentials for cluster that was just created
gcloud container clusters get-credentials e2e-test-cluster

# You may need to set your context if you have multiple clusters configured in kubeconfig
kubectl config set-cluster e2e-test-cluster

# Run the integration tests against cluster that was created
# Results will be printed to stdout
./e2e --test -- --kubeconfig ~/.kube/config

# Tear down the cluster
env PROJECT=<PROJECT> ZONE=<ZONE> ./e2e --down
```
