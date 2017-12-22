# Example Run #

## Prerequisites

- gcloud installed and configured in PATH
- ginkgo installed and configured in PATH

```bash
# Build the test binary by running the following in the e2e directory.
# The binary will be called e2e.test.
ginkgo build

# Create a cluster in the target account
# This will automatically retrieve the credentials for the cluster
# and append them to ~/.kube/config. This will also set your context
# to the e2e-test-cluster
env PROJECT=<PROJECT> ZONE=<ZONE> ./e2e.test --up

# Run the integration tests against cluster that was created
# Results will be printed to stdout
./e2e.test --kubeconfig ~/.kube/config --test

# Tear down the cluster
env PROJECT=<PROJECT> ZONE=<ZONE> ./e2e.test --down

# You can also do a run like this
# The order of the flags does not matter
env PROJECT=<PROJECT> ZONE=<ZONE> ./e2e.test --up --test --down
```
