package main

import (
	"context"
	"fmt"
	"os"
	"regexp"

	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ecr"
	"github.com/aws/aws-sdk-go/service/ecs"
)

func main() {
	lambda.Start(handler)
}

type event struct {
	Version string `json:"version"`
}

type response struct {
	TaskDefinition string
}

func mustGetEnv(name string) string {
	if value, ok := os.LookupEnv(name); ok {
		return value
	}
	panic("Can not found environment variable: " + name)
}

func handler(ctx context.Context, e event) (*response, error) {
	clusterName := mustGetEnv("ECS_CLUSTER")
	serviceName := mustGetEnv("ECS_SERVICE")
	taskdefName := mustGetEnv("ECS_TASKDEF")
	imageName := mustGetEnv("IMAGE_NAME")
	useImageDigest := os.Getenv("ECR_USE_IMAGE_DIGEST")

	sess := session.Must(session.NewSession())
	ecsClient := ecs.New(sess)
	ecrClient := ecr.New(sess)

	imagePrefix := ":" + e.Version

	match := regexp.MustCompile("^[^.]+\\.dkr\\.ecr\\.[^.]+\\.amazonaws\\.com/(.*)$").FindStringSubmatch(imageName)
	if len(match) >= 2 {
		imageRes, err := ecrClient.DescribeImagesWithContext(ctx, &ecr.DescribeImagesInput{
			RepositoryName: aws.String(match[1]),
			ImageIds: []*ecr.ImageIdentifier{
				&ecr.ImageIdentifier{ImageTag: aws.String(e.Version)},
			},
		})
		if err != nil {
			return nil, err
		}

		if useImageDigest == "true" {
			imagePrefix = "@" + aws.StringValue(imageRes.ImageDetails[0].ImageDigest)
		}
	}

	taskdefRes, err := ecsClient.DescribeTaskDefinitionWithContext(ctx, &ecs.DescribeTaskDefinitionInput{
		TaskDefinition: aws.String(taskdefName),
	})
	if err != nil {
		return nil, err
	}

	taskdef := taskdefRes.TaskDefinition

	firstContainer := taskdef.ContainerDefinitions[0]
	firstContainer.Image = aws.String(imageName + imagePrefix)
	if aws.StringValue(firstContainer.LogConfiguration.LogDriver) == "awslogs" {
		logStreamPrefix := fmt.Sprintf("%s/%s/", aws.StringValue(firstContainer.Name), e.Version)
		firstContainer.LogConfiguration.Options["awslogs-stream-prefix"] = aws.String(logStreamPrefix)
	}

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
		return nil, err
	}

	_, err = ecsClient.UpdateServiceWithContext(ctx, &ecs.UpdateServiceInput{
		Cluster:        aws.String(clusterName),
		Service:        aws.String(serviceName),
		TaskDefinition: newTaskdef.TaskDefinition.TaskDefinitionArn,
	})
	if err != nil {
		return nil, err
	}

	return &response{
		TaskDefinition: aws.StringValue(taskdefRes.TaskDefinition.TaskDefinitionArn),
	}, nil
}
