package platform

import (
	"context"
	"fmt"
	"reflect"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	infinimeshv1beta1 "github.com/infinimesh/operator/pkg/apis/infinimesh/v1beta1"
)

func (r *ReconcilePlatform) reconcileTimeseries(request reconcile.Request, instance *infinimeshv1beta1.Platform) error {
	log := logger.WithName("time series")
	{
		deploymentName := instance.Name + "-timescale-connector"

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
								Name:            "timescale-connector",
								Image:           "quay.io/infinimesh/timescale-connector:latest",
								ImagePullPolicy: corev1.PullAlways,
								EnvFrom: []corev1.EnvFromSource{
									{
										SecretRef: &corev1.SecretEnvSource{
											LocalObjectReference: corev1.LocalObjectReference{
												Name: instance.Name + "-timescaledb-auth",
											},
										},
									},
								},
								Env: []corev1.EnvVar{
									{
										Name:  "KAFKA_HOST",
										Value: instance.Spec.Kafka.BootstrapServers,
									},
									{
										Name:  "DB_ADDR",
										Value: "postgres://$(POSTGRES_USER):$(POSTGRES_PASSWORD)@" + instance.Name + "-timescaledb/postgres?sslmode=disable",
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
	}

	{
		pg := &unstructured.Unstructured{}
		pg.Object = map[string]interface{}{
			"kind":       "Postgres",
			"apiVersion": "kubedb.com/v1alpha1",
			"metadata": map[string]interface{}{
				"name":      instance.Name + "-timescaledb",
				"namespace": instance.Namespace,
			},
			"spec": map[string]interface{}{
				"version":     "11.1-v1",
				"storageType": "Durable",
				"storage": map[string]interface{}{
					"storageClassName": "standard",
					"accessModes":      []string{"ReadWriteOnce"},
					"resources": map[string]interface{}{
						"requests": map[string]interface{}{
							"storage": "50Gi",
						},
					},
				},
				"terminationPolicy": "DoNotTerminate",
			},
		}

		if err := controllerutil.SetControllerReference(instance, pg, r.scheme); err != nil {
			return err
		}

		foundPg := &unstructured.Unstructured{}
		foundPg.Object = map[string]interface{}{
			"apiVersion": "kubedb.com/v1alpha1",
			"kind":       "Postgres",
		}

		err := r.Get(context.TODO(), types.NamespacedName{Name: pg.GetName(), Namespace: pg.GetNamespace()}, foundPg)
		if err != nil && errors.IsNotFound(err) {
			log.Info("Creating Database", "namespace", pg.GetNamespace(), "name", pg.GetName())
			err = r.Create(context.TODO(), pg)
			if err != nil {
				return err
			}
		} else if err != nil {
			return err
		}
	}

	// Grafana
	{
		// TODO storage

		deploymentName := instance.Name + "-grafana"

		postgresSecret := &corev1.Secret{}
		err := r.Get(context.TODO(), types.NamespacedName{Name: instance.Name + "-timescaledb-auth", Namespace: instance.Namespace}, postgresSecret)
		if err != nil {
			log.Error(err, "Failed to get postgres credentials")
		}

		pw, ok := postgresSecret.Data["POSTGRES_PASSWORD"]
		if !ok {
			fmt.Println("Secret does not contain POSTGRES_PASSWORD")
		}

		ds := `apiVersion: 1
datasources:
- name: TimescaleDB
  isDefault: true
  type: postgres
  access: proxy
  orgId: 1
  url: %v
  user: postgres
  database: postgres
  jsonData:
    sslmode: "disable"
  secureJsonData:
    password: "%v"
  version: 1
  editable: false`

		// TODO use secure JSON data

		cm := &corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Name:      deploymentName + "-provision",
				Namespace: instance.Namespace,
			},
			Data: map[string]string{
				"global_timescaledb.yaml": fmt.Sprintf(ds, instance.Name+"-timescaledb", string(pw)),
			},
		}

		if err := controllerutil.SetControllerReference(instance, cm, r.scheme); err != nil {
			return err
		}

		foundCm := &corev1.ConfigMap{}
		err = r.Get(context.TODO(), types.NamespacedName{Name: cm.Name, Namespace: cm.Namespace}, foundCm)
		if err != nil && errors.IsNotFound(err) {
			err = r.Create(context.TODO(), cm)
			if err != nil {
				return err
			}
		} else if err != nil {
			return err
		} else {
			foundCm.Data = cm.Data
			err = r.Update(context.TODO(), foundCm)
			if err != nil {
				return err
			}
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
						Volumes: []corev1.Volume{
							{
								Name: "datasources",
								VolumeSource: corev1.VolumeSource{
									ConfigMap: &corev1.ConfigMapVolumeSource{
										LocalObjectReference: corev1.LocalObjectReference{
											Name: cm.Name,
										},
									},
								},
							},
						},
						Containers: []corev1.Container{
							{
								Name:  "grafana",
								Image: "grafana/grafana:latest",
								VolumeMounts: []corev1.VolumeMount{
									{
										Name:      "datasources",
										MountPath: "/etc/grafana/provisioning/datasources",
										ReadOnly:  true,
									},
								},
								Env: []corev1.EnvVar{
									{
										Name:  "GF_AUTH_PROXY_ENABLED",
										Value: "true",
									},
									{
										Name:  "GF_AUTH_PROXY_HEADER_NAME",
										Value: "X-WEBAUTH-USER",
									},
									{
										Name:  "GF_AUTH_PROXY_AUTO_SIGN_UP",
										Value: "true",
									},
									{
										Name:  "GF_AUTH_PROXY_HEADER_PROPERTY",
										Value: "username",
									},
									{
										Name:  "GF_USERS_AUTO_ASSIGN_ORG",
										Value: "true",
									},
									{
										Name:  "GF_USERS_AUTO_ASSIGN_ORG_ROLE",
										Value: "Viewer",
									},
									{
										Name:  "GF_ALERTING_ENABLED",
										Value: "false",
									},
								},
							},
							{
								Name:  "proxy",
								Image: "quay.io/infinimesh/grafana-proxy:latest",
								Env: []corev1.EnvVar{
									{
										Name:  "NODE_HOST",
										Value: instance.Name + "-nodeserver:8080",
									},
									{
										Name:  "GRAFANA_URL",
										Value: "http://" + instance.Name + "-grafana:3000",
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
			log.Info("Creating Grafana", "namespace", deploy.Namespace, "name", deploy.Name)
			err = r.Create(context.TODO(), deploy)
			if err != nil {
				return err
			}
		} else if err != nil {
			return err
		} else {
			if !reflect.DeepEqual(deploy.Spec, found.Spec) {
				found.Spec = deploy.Spec
				log.Info("Updating Grafana", "namespace", deploy.Namespace, "name", deploy.Name)
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
						Port:       3000,
						TargetPort: intstr.FromInt(3000),
					},
				},
			},
		}

		if err := controllerutil.SetControllerReference(instance, svc, r.scheme); err != nil {
			return err
		}

		foundSvc := &corev1.Service{}
		err = r.Get(context.TODO(), types.NamespacedName{Name: svc.Name, Namespace: svc.Namespace}, foundSvc)
		if err != nil && errors.IsNotFound(err) {
			err = r.Create(context.TODO(), svc)
			if err != nil {
				return err
			}
		} else if err != nil {
			return err
		}

		//TODO check if spec.apiserver.restful.tls exists
		// ingress := &extensionsv1beta1.Ingress{
		// 	ObjectMeta: metav1.ObjectMeta{
		// 		Name:      deploymentName,
		// 		Namespace: instance.Namespace,
		// 	},
		// 	Spec: extensionsv1beta1.IngressSpec{
		// 		TLS: instance.Spec.App.TLS,
		// 		Rules: []extensionsv1beta1.IngressRule{
		// 			{
		// 				Host: instance.Spec.App.Host,
		// 				IngressRuleValue: extensionsv1beta1.IngressRuleValue{
		// 					HTTP: &extensionsv1beta1.HTTPIngressRuleValue{
		// 						Paths: []extensionsv1beta1.HTTPIngressPath{
		// 							{
		// 								Backend: extensionsv1beta1.IngressBackend{
		// 									ServiceName: instance.Name + "-frontend",
		// 									ServicePort: intstr.FromInt(8080),
		// 								},
		// 							},
		// 						},
		// 					},
		// 				},
		// 			},
		// 		},
		// 	},
		// }

		// if err := controllerutil.SetControllerReference(instance, ingress, r.scheme); err != nil {
		// 	return err
		// }

		// foundIngress := &extensionsv1beta1.Ingress{}
		// err = r.Get(context.TODO(), types.NamespacedName{Name: ingress.Name, Namespace: ingress.Namespace}, foundIngress)
		// if err != nil && errors.IsNotFound(err) {
		// 	err = r.Create(context.TODO(), ingress)
		// 	if err != nil {
		// 		return err
		// 	}
		// } else if err != nil {
		// 	return err
		// }

	}

	return nil
}
