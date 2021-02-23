package platform

import (
	"context"
	"crypto/rand"
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

	"encoding/base64"

	infinimeshv1beta1 "github.com/infinimesh/operator/pkg/apis/infinimesh/v1beta1"
)

// GenerateRandomBytes returns securely generated random bytes.
// It will return an error if the system's secure random
// number generator fails to function correctly, in which
// case the caller should not continue.
func GenerateRandomBytes(n int) ([]byte, error) {
	b := make([]byte, n)
	_, err := rand.Read(b)
	// Note that err == nil only if we read len(b) bytes.
	if err != nil {
		return nil, err
	}

	return b, nil
}

func GenerateRandomKey(n int) (string, error) {
	randomKey, err := GenerateRandomBytes(32)
	if err != nil {
		return "", err
	}

	base64Secret := make([]byte, base64.StdEncoding.EncodedLen(len(randomKey)))
	return base64.StdEncoding.EncodeToString(base64Secret), nil
}

func (r *ReconcilePlatform) reconcileApiserver(request reconcile.Request, instance *infinimeshv1beta1.Platform) error {
	log := logger.WithName("apiserver")

	deploymentName := instance.Name + "-apiserver"

	randomKey, err := GenerateRandomBytes(32)
	if err != nil {
		return err
	}

	base64Secret := make([]byte, base64.StdEncoding.EncodedLen(len(randomKey)))
	base64.StdEncoding.Encode(base64Secret, []byte(randomKey))

	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      deploymentName,
			Namespace: instance.Namespace,
		},
		Data: map[string][]byte{
			"signing-key": base64Secret,
		},
	}

	if err := controllerutil.SetControllerReference(instance, secret, r.scheme); err != nil {
		return err
	}

	foundSecret := &corev1.Secret{}
	err = r.Get(context.TODO(), types.NamespacedName{Name: secret.Name, Namespace: secret.Namespace}, foundSecret)
	if err != nil && errors.IsNotFound(err) {
		log.Info("Creating Secret", "namespace", secret.Namespace, "name", secret.Name)
		err = r.Create(context.TODO(), secret)
		if err != nil {
			return err
		}
	} else if err != nil {
		return err
	}

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
							Name:            "apiserver",
							Image:           "quay.io/infinimesh/apiserver:latest",
							ImagePullPolicy: corev1.PullAlways,
							Env: []corev1.EnvVar{
								{
									Name:  "NODE_HOST",
									Value: instance.Name + "-nodeserver:8080",
								},
								{
									Name:  "REGISTRY_HOST",
									Value: instance.Name + "-device-registry:8080",
								},
								{
									Name:  "SHADOW_HOST",
									Value: instance.Name + "-shadow-api:8080",
								},
								{
									Name: "JWT_SIGNING_KEY",
									ValueFrom: &corev1.EnvVarSource{
										SecretKeyRef: &corev1.SecretKeySelector{
											LocalObjectReference: corev1.LocalObjectReference{
												Name: instance.Name + "-apiserver",
											},
											Key: "signing-key",
										},
									},
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
	err = r.Get(context.TODO(), types.NamespacedName{Name: deploy.Name, Namespace: deploy.Namespace}, found)
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
					TargetPort: intstr.FromInt(8080),
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

	ingress := &extensionsv1beta1.Ingress{
		ObjectMeta: metav1.ObjectMeta{
			Name:      deploymentName,
			Namespace: instance.Namespace,
			Annotations: map[string]string{
				"nginx.ingress.kubernetes.io/backend-protocol":   "GRPC",
				"nginx.ingress.kubernetes.io/proxy-read-timeout": "3600",
			},
		},
		Spec: extensionsv1beta1.IngressSpec{
			TLS: instance.Spec.Apiserver.GRPC.TLS,
			Rules: []extensionsv1beta1.IngressRule{
				{
					Host: instance.Spec.Apiserver.GRPC.Host,
					IngressRuleValue: extensionsv1beta1.IngressRuleValue{
						HTTP: &extensionsv1beta1.HTTPIngressRuleValue{
							Paths: []extensionsv1beta1.HTTPIngressPath{
								{
									Backend: extensionsv1beta1.IngressBackend{
										ServiceName: instance.Name + "-apiserver",
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
