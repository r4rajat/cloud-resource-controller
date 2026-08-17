package controller

import (
	"context"
	"errors"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	ec2types "github.com/aws/aws-sdk-go-v2/service/ec2/types"
	computev1 "github.com/r4rajat/cloud-resource-controller/api/v1"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

func createEC2Instance(ctx context.Context, ec2Instance *computev1.EC2Instance) (*computev1.CreatedEC2InstanceInfo, error) {
	logger := log.Log.WithName("createEC2Instance")

	logger.Info("Creating EC2 instance with the following specifications", "EC2Instance Name: ", ec2Instance.Name)

	ec2Client := awsClient(ec2Instance.Spec.Region)
	if ec2Client == nil {
		logger.Error(nil, "Failed to create AWS EC2 client")
		return nil, nil
	}

	runInput := &ec2.RunInstancesInput{
		ImageId:      aws.String(ec2Instance.Spec.AmiID),
		InstanceType: ec2types.InstanceType(ec2Instance.Spec.InstanceType),
		KeyName:      aws.String(ec2Instance.Spec.SSHKeyName),
		MinCount:     aws.Int32(1),
		MaxCount:     aws.Int32(1),
		SubnetId:     aws.String(ec2Instance.Spec.Subnet),
	}

	result, err := ec2Client.RunInstances(ctx, runInput)
	if err != nil {
		logger.Error(err, "Failed to create EC2 instance")
		return nil, errors.New("failed to create EC2 instance: " + err.Error())
	}

	if len(result.Instances) == 0 {
		logger.Error(nil, "No instances were created")
		return nil, nil
	}

	// Till here, the instance is created and we have
	// Instance IP, Private DNS & IP, Instance Type and Image ID
	instance := result.Instances[0]
	logger.Info("EC2 instance created successfully", "Instance ID: ", *instance.InstanceId)

	logger.Info("Waiting for the instance to be in 'running' state...")
	runWaiter := ec2.NewInstanceRunningWaiter(ec2Client)
	maxWaitTime := 3 * time.Minute

	describeInput := &ec2.DescribeInstancesInput{
		InstanceIds: []string{*instance.InstanceId},
	}

	err = runWaiter.Wait(ctx, describeInput, maxWaitTime)
	if err != nil {
		logger.Error(err, "Error while waiting for the instance to be in 'running' state")
		return nil, errors.New("error while waiting for the instance to be in 'running' state: " + err.Error())
	}

	logger.Info("Instance is now in 'running' state")
	logger.Info("Fetching instance details...")

	describeResult, err := ec2Client.DescribeInstances(ctx, describeInput)
	if err != nil {
		logger.Error(err, "Failed to describe EC2 instance")
		return nil, errors.New("failed to describe EC2 instance: " + err.Error())
	}

	logger.Info("Describe Result", "Public IP: ", describeResult.Reservations[0].Instances[0].PublicIpAddress, "State: ", describeResult.Reservations[0].Instances[0].State.Name)

	instance = describeResult.Reservations[0].Instances[0]
	createdInstanceInfo := &computev1.CreatedEC2InstanceInfo{
		InstanceID: *instance.InstanceId,
		PublicIP:   dereferenceString(instance.PublicIpAddress),
		PrivateIP:  dereferenceString(instance.PrivateIpAddress),
		PublicDNS:  dereferenceString(instance.PublicDnsName),
		PrivateDNS: dereferenceString(instance.PrivateDnsName),
		State:      string(instance.State.Name),
	}

	logger.Info("EC2 Instance Created Successfully...")
	return createdInstanceInfo, nil
}

func dereferenceString(s *string) string {
	if s == nil {
		return "<nil>"
	}
	return *s
}
