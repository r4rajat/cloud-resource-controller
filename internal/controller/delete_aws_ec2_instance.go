package controller

import (
	"context"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	computev1 "github.com/r4rajat/cloud-resource-controller/api/v1"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

func deleteEC2Instance(ctx context.Context, ec2InstanceObject *computev1.EC2Instance) error {
	logger := log.Log.WithName("deleteEC2Instance")

	logger.Info("Deleting EC2 instance with the following specifications", "name", ec2InstanceObject.Name)

	ec2Client := awsClient(ec2InstanceObject.Spec.Region)
	if ec2Client == nil {
		logger.Error(nil, "Failed to create AWS EC2 client")
		return nil
	}

	terminateInput := &ec2.TerminateInstancesInput{
		InstanceIds:    []string{ec2InstanceObject.Status.InstanceID},
		SkipOsShutdown: aws.Bool(true), // Skip OS shutdown for faster termination
	}

	terminateOutput, err := ec2Client.TerminateInstances(ctx, terminateInput)
	if err != nil {
		logger.Error(err, "Failed to terminate EC2 instance")
		return err
	}

	if len(terminateOutput.TerminatingInstances) == 0 {
		logger.Error(nil, "No instances were terminated")
		return nil
	}

	logger.Info("EC2 instance termination initiated successfully", "instanceID", terminateOutput.TerminatingInstances[0].InstanceId)

	logger.Info("Waiting for the instance to be in 'terminated' state...")

	deleteWaiter := ec2.NewInstanceTerminatedWaiter(ec2Client)
	maxWaitTime := 5 * time.Minute

	describeInput := &ec2.DescribeInstancesInput{
		InstanceIds: []string{ec2InstanceObject.Status.InstanceID},
	}

	err = deleteWaiter.Wait(ctx, describeInput, maxWaitTime)
	if err != nil {
		logger.Error(err, "Failed to wait for EC2 instance termination")
		return err
	}

	logger.Info("EC2 instance terminated successfully", "instanceID", terminateOutput.TerminatingInstances[0].InstanceId)

	return nil
}
