# Cloud Resource Controller

A Kubernetes controller built with Kubebuilder that manages AWS EC2 instances through a custom resource named `EC2Instance`.

This project creates and reconciles EC2 instances based on a declarative Kubernetes spec, tracks lifecycle state in the CR status, and cleans up the AWS resource when the custom resource is deleted.

## What this project does

- Watches resources of kind `EC2Instance`
- Creates EC2 instances in the configured AWS region
- Stores instance metadata in status fields such as `instanceID`, `state`, `publicIP`, and `privateIP`
- Adds a finalizer so AWS cleanup happens before resource deletion
- Uses controller-runtime and Kubebuilder scaffolding for reconciliation

## Project structure

- `api/v1/` contains the CRD schema and status definitions
- `internal/controller/` contains reconciliation and AWS logic
- `config/samples/` contains example custom resource manifests
- `Makefile` contains build, test, manifest generation, and deployment commands

## Prerequisites

Before running this project, make sure you have:

- Go 1.23+ (check with `go version`)
- Docker or Podman for image builds
- `kubectl` configured to a Kubernetes cluster
- AWS credentials available to the environment or the controller runtime
- Optional: `kind` if you want to run e2e tests locally

For AWS access, the controller uses the standard AWS SDK environment and credentials chain, so either:

- export `AWS_ACCESS_KEY_ID`, `AWS_SECRET_ACCESS_KEY`, and optionally `AWS_SESSION_TOKEN`
- or use a configured AWS profile / IAM role

## CRD fields

The custom resource is defined under `compute.r4rajat.com/v1` and has the following core spec:

```yaml
apiVersion: compute.r4rajat.com/v1
kind: EC2Instance
metadata:
  name: ec2instance-sample
spec:
  instanceName: "my-ec2-instance"
  amiID: "ami-0bdc7d025135d7b49"
  sshKeyName: "my-rsa-keypair"
  instanceType: "t3.micro"
  subnet: "subnet-0521abfda996463f5"
  region: "us-east-1"
  associatePublicIP: true
  securityGroups:
    - sg-04d98c33d289f2dc5
  tags:
    environment: "dev"
    owner: "r4rajat"
  storage:
    rootVolume:
      size: 10
      type: "gp2"
```

The status fields are updated after provisioning and include:

- `instanceID`
- `state`
- `publicIP`
- `privateIP`
- `publicDNS`
- `privateDNS`

## Local development

Generate the CRD and DeepCopy code:

```bash
make manifests
make generate
```

Run the controller locally against your current kubeconfig context:

```bash
make run
```

Run formatting and validation:

```bash
make fmt
make vet
```

Run unit tests:

```bash
make test
```

## Build and deploy

Build the manager binary:

```bash
make build
```

Build the container image:

```bash
export IMG=your-registry/cloud-resource-controller:latest
make docker-build IMG=$IMG
```

Push the image:

```bash
make docker-push IMG=$IMG
```

Deploy to the cluster:

```bash
make deploy IMG=$IMG
```

Install CRDs only:

```bash
make install
```

## Example resource

A sample resource is provided at `config/samples/compute_v1_ec2instance.yaml`.

Apply it with:

```bash
kubectl apply -f config/samples/compute_v1_ec2instance.yaml
```

Then inspect the resource:

```bash
kubectl get ec2instances
kubectl describe ec2instance ec2instance-sample
```

## Controller behavior

The reconciler does the following:

1. Reads the `EC2Instance` resource from the cluster.
2. Checks whether it was deleted and, if so, tears down the AWS instance before removing the finalizer.
3. Skips creation if the resource already has an `instanceID` in status.
4. Adds a finalizer for safe cleanup.
5. Calls AWS to create the EC2 instance.
6. Updates the CR status with the instance metadata returned by AWS.

## Useful Make targets

```bash
make help
make manifests
make generate
make test
make run
make build
make deploy IMG=<image>
make undeploy
```

## Notes

- This controller is designed for a single EC2 instance resource per CR.
- The underlying AWS instance is created with the fields from `spec`.
- If you change the API types under `api/v1`, regenerate manifests and generated code with `make manifests` and `make generate`.

## License

This project is licensed under the Apache License 2.0. See the license header in the source files for details.
