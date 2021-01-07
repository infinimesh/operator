package platform

import (
	"context"
	"reflect"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"

	extensionsv1beta1 "k8s.io/api/extensions/v1beta1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	infinimeshv1beta1 "github.com/infinimesh/operator/pkg/apis/infinimesh/v1beta1"
)

func (r *ReconcilePlatform) reconcileApiserverRest(request reconcile.Request, instance *infinimeshv1beta1.Platform) error {
	log := logger.WithName("apiserver-rest")

	deploymentName := instance.Name + "-apiserver-rest"

	deploy := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      deploymentName,
			Namespace: instance.Namespace,
		},
		Spec: appsv1.DeploymentSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{"deployment": deploymentName},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{Labels: map[string]string{"deployment": deploymentName}},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:            "apiserver-rest",
							Image:           "quay.io/infinimesh/apiserver-rest:latest",
							ImagePullPolicy: corev1.PullAlways,
							Env: []corev1.EnvVar{
								{
									Name:  "APISERVER_ENDPOINT",
									Value: instance.Name + "-apiserver:8080",
								},
							},
						},
					},
				},
			},
		},
	}

	if err := controllerutil.SetControllerReference(instance, deploy, r.scheme); err != nil {
		return err
	}

	found := &appsv1.Deployment{}
	err := r.Get(context.TODO(), types.NamespacedName{Name: deploy.Name, Namespace: deploy.Namespace}, found)
	if err != nil && errors.IsNotFound(err) {
		log.Info("Creating Deployment", "namespace", deploy.Namespace, "name", deploy.Name)
		err = r.Create(context.TODO(), deploy)
		if err != nil {
			return err
		}
	} else if err != nil {
		return err
	} else {
		if !reflect.DeepEqual(deploy.Spec, found.Spec) {
			found.Spec = deploy.Spec
			log.Info("Updating Deployment", "namespace", deploy.Namespace, "name", deploy.Name)
			err = r.Update(context.TODO(), found)
			if err != nil {
				return err
			}
		}

	}

	svc := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      deploymentName,
			Namespace: instance.Namespace,
		},
		Spec: corev1.ServiceSpec{
			Selector: map[string]string{"deployment": deploymentName},
			Type:     corev1.ServiceTypeClusterIP,
			Ports: []corev1.ServicePort{
				{
					Protocol:   corev1.ProtocolTCP,
					Port:       8080,
					TargetPort: intstr.FromInt(8081),
				},
			},
		},
	}

	if err := controllerutil.SetControllerReference(instance, svc, r.scheme); err != nil {
		return err
	}

	foundSvc := &corev1.Service{}
	err = r.Get(context.TODO(), types.NamespacedName{Name: deploy.Name, Namespace: deploy.Namespace}, foundSvc)
	if err != nil && errors.IsNotFound(err) {
		err = r.Create(context.TODO(), svc)
		if err != nil {
			return err
		}
	} else if err != nil {
		return err
	}

	//TODO check if spec.apiserver.restful.tls exists
	ingress := &extensionsv1beta1.Ingress{
		ObjectMeta: metav1.ObjectMeta{
			Name:      deploymentName,
			Namespace: instance.Namespace,
			Annotations: map[string]string{
				"nginx.ingress.kubernetes.io/proxy-read-timeout": "3600",
			},
		},
		Spec: extensionsv1beta1.IngressSpec{
			TLS: instance.Spec.Apiserver.Restful.TLS,
			Rules: []extensionsv1beta1.IngressRule{
				{
					Host: instance.Spec.Apiserver.Restful.Host,
					IngressRuleValue: extensionsv1beta1.IngressRuleValue{
						HTTP: &extensionsv1beta1.HTTPIngressRuleValue{
							Paths: []extensionsv1beta1.HTTPIngressPath{
								{
									Backend: extensionsv1beta1.IngressBackend{
										ServiceName: instance.Name + "-apiserver-rest",
										ServicePort: intstr.FromInt(8080),
									},
								},
							},
						},
					},
				},
			},
		},
	}

	if err := controllerutil.SetControllerReference(instance, ingress, r.scheme); err != nil {
		return err
	}

	foundIngress := &extensionsv1beta1.Ingress{}
	err = r.Get(context.TODO(), types.NamespacedName{Name: ingress.Name, Namespace: ingress.Namespace}, foundIngress)
	if err != nil && errors.IsNotFound(err) {
		err = r.Create(context.TODO(), ingress)
		if err != nil {
			return err
		}
	} else if err != nil {
		return err
	}

	return nil

}
