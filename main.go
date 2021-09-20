// Source https://www.pulumi.com/docs/guides/crosswalk/aws/eks/

package main

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/pulumi/pulumi-aws/sdk/v4/go/aws/ecr"
	"github.com/pulumi/pulumi-docker/sdk/v3/go/docker"
	"github.com/pulumi/pulumi-eks/sdk/go/eks"
	k8s "github.com/pulumi/pulumi-kubernetes/sdk/v3/go/kubernetes"
	appsv1 "github.com/pulumi/pulumi-kubernetes/sdk/v3/go/kubernetes/apps/v1"
	corev1 "github.com/pulumi/pulumi-kubernetes/sdk/v3/go/kubernetes/core/v1"
	metav1 "github.com/pulumi/pulumi-kubernetes/sdk/v3/go/kubernetes/meta/v1"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

func main() {
	pulumi.Run(func(ctx *pulumi.Context) error {
		// Create an EKS cluster with the default configuration.
		cluster, err := eks.NewCluster(ctx, "KubPoc-1", nil)
		if err != nil {
			return err
		}

		// Create a Kubernetes provider using the new cluster's Kubeconfig.
		provider, err := k8s.NewProvider(ctx, "eksProvider", &k8s.ProviderArgs{
			Kubeconfig: cluster.Kubeconfig.ApplyT(
				func(config interface{}) (string, error) {
					b, err := json.Marshal(config)
					if err != nil {
						return "", err
					}
					return string(b), nil
				}).(pulumi.StringOutput),
		})
		if err != nil {
			return err
		}

		repo, err := ecr.NewRepository(ctx, "my-repo", nil)
		if err != nil {
			return err
		}

		// ...2) Get registry info (creds and endpoint).
		imageName := repo.RepositoryUrl
		registryInfo := repo.RegistryId.ApplyT(func(id string) (docker.ImageRegistry, error) {
			creds, err := ecr.GetCredentials(ctx, &ecr.GetCredentialsArgs{RegistryId: id})
			if err != nil {
				return docker.ImageRegistry{}, err
			}
			decoded, err := base64.StdEncoding.DecodeString(creds.AuthorizationToken)
			if err != nil {
				return docker.ImageRegistry{}, err
			}
			parts := strings.Split(string(decoded), ":")
			if len(parts) != 2 {
				return docker.ImageRegistry{}, errors.New("Invalid credentials")
			}
			return docker.ImageRegistry{
				Server:   creds.ProxyEndpoint,
				Username: parts[0],
				Password: parts[1],
			}, nil
		}).(docker.ImageRegistryOutput)

		// ...3) Build and publish the container image.
		image, err := docker.NewImage(ctx, "secure-nginx-app", &docker.ImageArgs{
			Build:     &docker.DockerBuildArgs{Context: pulumi.String("app")},
			ImageName: imageName,
			Registry:  registryInfo,
		})

		// Declare a deployment that targets this provider:
		appName := "my-app"
		appLabels := pulumi.StringMap{"app": pulumi.String(appName)}
		_, err = appsv1.NewDeployment(ctx,
			fmt.Sprintf("%s-dep", appName),
			&appsv1.DeploymentArgs{
				Spec: &appsv1.DeploymentSpecArgs{
					Selector: &metav1.LabelSelectorArgs{MatchLabels: appLabels},
					Replicas: pulumi.Int(2),
					Template: &corev1.PodTemplateSpecArgs{
						Metadata: &metav1.ObjectMetaArgs{Labels: appLabels},
						Spec: &corev1.PodSpecArgs{
							Containers: corev1.ContainerArray{
								&corev1.ContainerArgs{
									Name:  pulumi.String(appName),
									Image: image.ImageName,
								},
							},
						},
					},
				},
			},
			// Use our custom provider for this object.
			pulumi.Provider(provider),
		)
		if err != nil {
			return nil
		}

		service, err := corev1.NewService(ctx,
			fmt.Sprintf("%s-svc", appName),
			&corev1.ServiceArgs{
				Spec: &corev1.ServiceSpecArgs{
					Type:     pulumi.String("LoadBalancer"),
					Selector: appLabels,
					Ports: corev1.ServicePortArray{
						&corev1.ServicePortArgs{
							Name:     pulumi.String("http"),
							Port:     pulumi.Int(80),
							Protocol: pulumi.String("TCP"),
						},
						&corev1.ServicePortArgs{
							Name:     pulumi.String("https"),
							Port:     pulumi.Int(443),
							Protocol: pulumi.String("TCP"),
						},
					},
				},
			},
			pulumi.Provider(provider),
		)
		if err != nil {
			return err
		}

		// Export the URL for the load balanced service.
		ctx.Export("url", service.Status.ApplyT(func(status interface{}) string {
			return *status.(*corev1.ServiceStatus).LoadBalancer.Ingress[0].Hostname
		}))

		// Export the cluster's kubeconfig.
		ctx.Export("kubeconfig", cluster.Kubeconfig)
		return nil
	})
}
