package main

import (
	"context"
	"os"
	"regexp"

	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ecr"
	"github.com/aws/aws-sdk-go/service/ecs"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
)

func main() {
	log.SetFormatter(&log.JSONFormatter{})
	lambda.Start(handler)
}

type event struct {
	Version string `json:"version"`
}

type response struct {
	TaskDefinition string
	Deployment     []*ecs.Deployment
}

var (
	awsSession = session.Must(session.NewSession())
	ecsClient  = ecs.New(awsSession)
	ecrClient  = ecr.New(awsSession)
)

func handler(ctx context.Context, e event) (*response, error) {
	clusterName := mustGetEnv("ECS_CLUSTER")
	serviceName := mustGetEnv("ECS_SERVICE")
	taskdefName := mustGetEnv("ECS_TASKDEF")
	imageName := mustGetEnv("IMAGE_NAME")
	useImageDigest := os.Getenv("ECR_USE_IMAGE_DIGEST")

	log.WithField("version", e.Version).Info("Start deployment")

	// by default use version as image tag
	imageSuffix := ":" + e.Version

	// check if image stored on ECR
	match := regexp.MustCompile("^([^.]+)\\.dkr\\.ecr\\.[^.]+\\.amazonaws\\.com/(.*)$").FindStringSubmatch(imageName)
	if len(match) >= 3 {
		log.
			WithField("accountId", match[1]).
			WithField("repoName", match[2]).
			Info("Checking docker image")
		imageRes, err := ecrClient.DescribeImagesWithContext(ctx, &ecr.DescribeImagesInput{
			RegistryId:     aws.String(match[1]),
			RepositoryName: aws.String(match[2]),
			ImageIds: []*ecr.ImageIdentifier{
				&ecr.ImageIdentifier{ImageTag: aws.String(e.Version)},
			},
		})
		if err != nil {
			log.WithError(err).Error("Can not describe image")
			return nil, errors.Wrapf(err, "Can not describe image %s:%s", imageName, e.Version)
		}

		log.
			WithField("digest", aws.StringValue(imageRes.ImageDetails[0].ImageDigest)).
			WithField("tag", e.Version).
			Info("Image found")

		if useImageDigest == "true" {
			imageSuffix = "@" + aws.StringValue(imageRes.ImageDetails[0].ImageDigest)
		}
	} else {
		log.WithField("name", imageName).Warn("Image name is not ECR repository")
	}

	// get latest task defintion
	taskdefRes, err := ecsClient.DescribeTaskDefinitionWithContext(ctx, &ecs.DescribeTaskDefinitionInput{
		TaskDefinition: aws.String(taskdefName),
	})
	if err != nil {
		log.WithError(err).Error("Can not describe task definition")
		return nil, errors.Wrapf(err, "Can not describe task definition %s", taskdefName)
	}

	// update container's image
	taskdef := taskdefRes.TaskDefinition
	firstContainer := taskdef.ContainerDefinitions[0]
	firstContainer.Image = aws.String(imageName + imageSuffix)

	// update log stream prefix
	if aws.StringValue(firstContainer.LogConfiguration.LogDriver) == "awslogs" {
		firstContainer.LogConfiguration.Options["awslogs-stream-prefix"] = aws.String(e.Version)
	}

	// copy task defintion
	taskDefInput := &ecs.RegisterTaskDefinitionInput{
		ContainerDefinitions:    taskdef.ContainerDefinitions,
		Cpu:                     taskdef.Cpu,
		ExecutionRoleArn:        taskdef.ExecutionRoleArn,
		Family:                  taskdef.Family,
		IpcMode:                 taskdef.IpcMode,
		Memory:                  taskdef.Memory,
		NetworkMode:             taskdef.NetworkMode,
		PidMode:                 taskdef.PidMode,
		PlacementConstraints:    taskdef.PlacementConstraints,
		ProxyConfiguration:      taskdef.ProxyConfiguration,
		RequiresCompatibilities: taskdef.RequiresCompatibilities,
		Tags:                    taskdefRes.Tags,
		TaskRoleArn:             taskdef.TaskRoleArn,
		Volumes:                 taskdef.Volumes,
	}

	newTaskdef, err := ecsClient.RegisterTaskDefinitionWithContext(ctx, taskDefInput)
	if err != nil {
		log.WithError(err).Error("Can not register new task definition")
		return nil, errors.Wrap(err, "Can not register new task definition")
	}

	// the actual update
	log.
		WithField("cluster", clusterName).
		WithField("service", serviceName).
		WithField("taskdef", aws.StringValue(newTaskdef.TaskDefinition.TaskDefinitionArn)).
		Info("Updating ECS service")
	service, err := ecsClient.UpdateServiceWithContext(ctx, &ecs.UpdateServiceInput{
		Cluster:        aws.String(clusterName),
		Service:        aws.String(serviceName),
		TaskDefinition: newTaskdef.TaskDefinition.TaskDefinitionArn,
	})
	if err != nil {
		log.WithError(err).Error("Can not update service")
		return nil, errors.Wrapf(
			err,
			"Can not update service '%s' with new task definition '%s'",
			serviceName,
			aws.StringValue(newTaskdef.TaskDefinition.TaskDefinitionArn),
		)
	}

	log.WithField("deployment", service.Service.Deployments).Info("Service updated")
	return &response{
		TaskDefinition: aws.StringValue(service.Service.TaskDefinition),
		Deployment:     service.Service.Deployments,
	}, nil
}

func mustGetEnv(name string) string {
	if value, ok := os.LookupEnv(name); ok {
		return value
	}
	panic("Can not found environment variable: " + name)
}
