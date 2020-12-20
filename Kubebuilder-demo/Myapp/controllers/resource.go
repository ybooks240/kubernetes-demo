package controllers

import (
	appv1 "github.com/ybooks240/api/v1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// NewDeploy ...
func NewDeploy(app *appv1.Myapp) *appsv1.Deployment {
	labels := map[string]string{"myapp": app.Name}
	selector := &metav1.LabelSelector{
		MatchLabels: labels,
	}
	return &appsv1.Deployment{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "apps/v1",
			Kind:       "Deployment",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      app.Name,
			Namespace: app.Namespace,
			// OwnerReference
			OwnerReferences: makeOwnerReferences(app),
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: app.Spec.Size,
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: labels,
				},
				Spec: corev1.PodSpec{
					Containers: newContainers(app),
				},
			},
			Selector: selector,
		},
	}
}

func makeOwnerReferences(app *appv1.Myapp) []metav1.OwnerReference {
	return []metav1.OwnerReference{
		*metav1.NewControllerRef(app, schema.GroupVersionKind{
			Kind:    appv1.Kind,
			Group:   appv1.GroupVersion.Group,
			Version: appv1.GroupVersion.Version,
		}),
	}
}

func newContainers(app *appv1.Myapp) []corev1.Container {
	containerPort := []corev1.ContainerPort{}
	for _, svcPort := range app.Spec.Ports {
		containerPort = append(containerPort, corev1.ContainerPort{
			ContainerPort: svcPort.TargetPort.IntVal,
		})
	}
	return []corev1.Container{
		{
			Name:  app.Name,
			Image: app.Spec.Image,
			//Resources: app.Spec.Resources,
			Ports: containerPort,
		},
	}
}

func NewService(app *appv1.Myapp) *corev1.Service {
	return &corev1.Service{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Service",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:            app.Name,
			Namespace:       app.Namespace,
			OwnerReferences: makeOwnerReferences(app),
		},
		Spec: corev1.ServiceSpec{
			Ports: app.Spec.Ports,
			Type:  corev1.ServiceTypeNodePort,
			Selector: map[string]string{
				"myapp": app.Name,
			},
		},
	}
}

// TODO 更新操作
