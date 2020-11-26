package platform

import (
	"context"

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

func (r *ReconcilePlatform) reconcileDeviceDetails(request reconcile.Request, instance *infinimeshv1beta1.Platform) error {
	log := logger.WithName("Redis Device Details")

	replicas := int32(3)

	svc := &corev1.Service{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Service",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      instance.Name + "-redis-device-details",
			Namespace: instance.Namespace,
			Labels: map[string]string{
				"app": instance.Name + "-redis-device-details",
			},
		},
		Spec: corev1.ServiceSpec{
			ClusterIP: "None",
			Type:      corev1.ServiceTypeClusterIP,
			Selector:  map[string]string{"app": instance.Name + "-redis-device-details"},
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

	found := &corev1.Service{}
	err := r.Get(context.TODO(), types.NamespacedName{Name: svc.Name, Namespace: svc.Namespace}, found)
	if err != nil && errors.IsNotFound(err) {
		log.Info("Creating device details service", "namespace", svc.Namespace, "name", svc.Name)
		err = r.Create(context.TODO(), svc)
		if err != nil {
			return err
		}
	} else if err != nil {
		return err
	}

	var pvcSpec corev1.PersistentVolumeClaimSpec
	if instance.Spec.DGraph.Storage == nil {
		pvcSpec = corev1.PersistentVolumeClaimSpec{
			AccessModes: []corev1.PersistentVolumeAccessMode{corev1.ReadWriteOnce},
			Resources: corev1.ResourceRequirements{
				Requests: corev1.ResourceList{corev1.ResourceStorage: resource.MustParse("1Gi")},
			},
		}
	} else {
		pvcSpec = *instance.Spec.DGraph.Storage
	}

	statefulSetDeviceDetails := &appsv1.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      instance.Name + "-redis-device-details",
			Namespace: instance.Namespace,
		},
		Spec: appsv1.StatefulSetSpec{
			ServiceName: instance.Name + "-redis-device-details",
			Replicas:    &replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{"app": instance.Name + "-redis-device-details"}, // TODO
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{Labels: map[string]string{"app": instance.Name + "-redis-device-details"}},
				Spec: corev1.PodSpec{
					Affinity: &corev1.Affinity{
						PodAntiAffinity: &corev1.PodAntiAffinity{
							PreferredDuringSchedulingIgnoredDuringExecution: []corev1.WeightedPodAffinityTerm{
								{
									Weight: 100,
									PodAffinityTerm: corev1.PodAffinityTerm{
										LabelSelector: &metav1.LabelSelector{
											MatchExpressions: []metav1.LabelSelectorRequirement{
												{
													Key:      "app",
													Operator: metav1.LabelSelectorOpIn,
													Values: []string{
														instance.Name + "-redis-device-details",
													},
												},
											},
										},
										TopologyKey: "kubernetes.io/hostname",
									},
								},
							},
						},
					},
					Containers: []corev1.Container{
						{
							Name:            "redis-device-details",
							Image:           "redis:5.0.10",
							ImagePullPolicy: corev1.PullAlways,
							Ports: []corev1.ContainerPort{
								{
									ContainerPort: 6379,
									Name:          "web",
								},
							},
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:      "datadir",
									MountPath: "/data",
								},
							},

							Command: []string{
								"redis-server", "--appendonly", "yes",
							},
						},
					},
					TerminationGracePeriodSeconds: func() *int64 { val := int64(60); return &val }(),
					Volumes: []corev1.Volume{
						{
							Name: "datadir",
							VolumeSource: corev1.VolumeSource{
								PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
									ClaimName: "datadir",
								},
							},
						},
					},
				},
			},
			UpdateStrategy: appsv1.StatefulSetUpdateStrategy{
				Type: "RollingUpdate",
			},
			VolumeClaimTemplates: []corev1.PersistentVolumeClaim{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "datadir",
					},
					Spec: pvcSpec,
				},
			},
		},
	}

	if err := controllerutil.SetControllerReference(instance, statefulSetDeviceDetails, r.scheme); err != nil {
		return err
	}

	foundS := &appsv1.StatefulSet{}
	err = r.Get(context.TODO(), types.NamespacedName{Name: statefulSetDeviceDetails.Name, Namespace: statefulSetDeviceDetails.Namespace}, foundS)
	if err != nil && errors.IsNotFound(err) {
		log.Info("Creating Device Details statefulsSet", "namespace", statefulSetDeviceDetails.Namespace, "name", statefulSetDeviceDetails.Name)
		err = r.Create(context.TODO(), statefulSetDeviceDetails)
		if err != nil {
			return err
		}
	} else if err != nil {
		return err
	}

	return nil
}
