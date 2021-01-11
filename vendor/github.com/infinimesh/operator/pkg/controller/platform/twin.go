package platform

import (
	"context"
	"reflect"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	infinimeshv1beta1 "github.com/infinimesh/operator/pkg/apis/infinimesh/v1beta1"
)

func (r *ReconcilePlatform) reconcileTwin(request reconcile.Request, instance *infinimeshv1beta1.Platform) error {
	log := logger.WithName("twin")
	{
		deploymentName := instance.Name + "-shadow-delta-merger"
		// TODO(user): Change this to be the object type created by your controller
		// Define the desired Deployment object

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
								Name:            "shadow-delta-merger",
								Image:           "quay.io/infinimesh/shadow-delta-merger:latest",
								ImagePullPolicy: corev1.PullAlways,
								Env: []corev1.EnvVar{
									{
										Name:  "KAFKA_HOST",
										Value: instance.Spec.Kafka.BootstrapServers,
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
		deploymentName := instance.Name + "-shadow-persister"
		// TODO(user): Change this to be the object type created by your controller
		// Define the desired Deployment object

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
								Name:            "shadow-persister",
								Image:           "quay.io/infinimesh/shadow-persister:latest",
								ImagePullPolicy: corev1.PullAlways,
								Env: []corev1.EnvVar{
									{
										Name:  "KAFKA_HOST",
										Value: instance.Spec.Kafka.BootstrapServers,
									},
									{
										Name:  "DB_ADDR",
										Value: instance.Name + "-twin-redis:6379",
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
		deploymentName := instance.Name + "-shadow-api"
		// TODO(user): Change this to be the object type created by your controller
		// Define the desired Deployment object

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
								Name:            "shadow-api",
								Image:           "quay.io/infinimesh/shadow-api:latest",
								ImagePullPolicy: corev1.PullAlways,
								Env: []corev1.EnvVar{
									{
										Name:  "KAFKA_HOST",
										Value: instance.Spec.Kafka.BootstrapServers,
									},
									{
										Name:  "DB_ADDR",
										Value: instance.Name + "-twin-redis:6379",
									},
									{
										Name:  "DEVICE_REGISTRY_URL",
										Value: instance.Name + "-device-registry:8080",
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
						TargetPort: intstr.FromInt(8096),
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

	}

	{
		deploymentName := instance.Name + "-twin-redis"

		redisS := &appsv1.StatefulSet{
			ObjectMeta: metav1.ObjectMeta{
				Name:      deploymentName,
				Namespace: instance.Namespace,
			},
			Spec: appsv1.StatefulSetSpec{
				VolumeClaimTemplates: []corev1.PersistentVolumeClaim{
					{
						ObjectMeta: metav1.ObjectMeta{
							Labels: map[string]string{"deployment": deploymentName},
							Name:   "redis-data",
						},
						Spec: corev1.PersistentVolumeClaimSpec{
							AccessModes: []corev1.PersistentVolumeAccessMode{corev1.ReadWriteOnce},
							Resources: corev1.ResourceRequirements{
								Requests: corev1.ResourceList{corev1.ResourceStorage: resource.MustParse("5Gi")},
							},
						},
					},
				},
				Selector: &metav1.LabelSelector{
					MatchLabels: map[string]string{"deployment": deploymentName},
				},
				Template: corev1.PodTemplateSpec{
					ObjectMeta: metav1.ObjectMeta{Labels: map[string]string{"deployment": deploymentName}},
					Spec: corev1.PodSpec{
						Containers: []corev1.Container{
							{
								Name:  "redis",
								Image: "redis:latest",
								Env:   []corev1.EnvVar{},
								VolumeMounts: []corev1.VolumeMount{
									{
										Name:      "redis-data",
										MountPath: "/data",
									},
								},
							},
						},
					},
				},
			},
		}

		if err := controllerutil.SetControllerReference(instance, redisS, r.scheme); err != nil {
			return err
		}

		foundRedis := &appsv1.StatefulSet{}
		err := r.Get(context.TODO(), types.NamespacedName{Name: redisS.Name, Namespace: redisS.Namespace}, foundRedis)
		if err != nil && errors.IsNotFound(err) {
			log.Info("Creating statefulset", "namespace", redisS.Namespace, "name", redisS.Name)
			err = r.Create(context.TODO(), redisS)
			if err != nil {
				return err
			}
		} else if err != nil {
			return err
		} else {
			if !reflect.DeepEqual(redisS.Spec, foundRedis.Spec) {
				foundRedis.Spec = redisS.Spec
				log.Info("Updating statefulset", "namespace", redisS.Namespace, "name", redisS.Name)
				err = r.Update(context.TODO(), foundRedis)
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
						Port:       6379,
						TargetPort: intstr.FromInt(6379),
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

	}

	return nil
}
