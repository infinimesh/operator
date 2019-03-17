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

func (r *ReconcilePlatform) reconcileDgraph(request reconcile.Request, instance *infinimeshv1beta1.Platform) error {
	log := logger.WithName("dgraph")

	replicas := int32(3)

	svc := &corev1.Service{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Service",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      instance.Name + "-dgraph-zero",
			Namespace: instance.Namespace,
			Labels: map[string]string{
				"app": instance.Name + "-dgraph-zero",
			},
		},
		Spec: corev1.ServiceSpec{
			ClusterIP: "None",
			Selector:  map[string]string{"app": instance.Name + "-dgraph-zero"},
			Ports: []corev1.ServicePort{
				{
					Port:       5080,
					TargetPort: intstr.FromInt(5080),
					Name:       "zero-grpc",
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
		log.Info("Creating zero service", "namespace", svc.Namespace, "name", svc.Name)
		err = r.Create(context.TODO(), svc)
		if err != nil {
			return err
		}
	} else if err != nil {
		return err
	}

	statefulSetZero := &appsv1.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      instance.Name + "-dgraph-zero",
			Namespace: instance.Namespace,
		},
		Spec: appsv1.StatefulSetSpec{
			ServiceName: instance.Name + "-dgraph-zero",
			Replicas:    &replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{"app": instance.Name + "-dgraph-zero"}, // TODO
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{Labels: map[string]string{"app": instance.Name + "-dgraph-zero"}},
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
														instance.Name + "-dgraph-zero",
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
							Name:            "zero",
							Image:           "dgraph/dgraph:latest",
							ImagePullPolicy: corev1.PullAlways,
							Ports: []corev1.ContainerPort{
								{
									ContainerPort: 5080,
									Name:          "zero-grpc",
								},
								{
									ContainerPort: 6080,
									Name:          "zero-http",
								},
							},
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:      "datadir",
									MountPath: "/dgraph",
								},
							},
							Env: []corev1.EnvVar{
								{
									Name: "POD_NAMESPACE",
									ValueFrom: &corev1.EnvVarSource{
										FieldRef: &corev1.ObjectFieldSelector{
											FieldPath: "metadata.namespace",
										},
									},
								},
							},
							Command: []string{
								"bash",
								"-c",
								`set -ex
[[ ` + "`hostname`" + ` =~ -([0-9]+)$ ]] || exit 1
ordinal=${BASH_REMATCH[1]}
idx=$(($ordinal + 1))
if [[ $ordinal -eq 0 ]]; then
dgraph zero --my=$(hostname -f):5080 --idx $idx --replicas 3
else
dgraph zero --my=$(hostname -f):5080 --peer ` + instance.Name + `-dgraph-zero-0.` + instance.Name + `-dgraph-zero.${POD_NAMESPACE}.svc.cluster.local:5080 --idx $idx --replicas 3
fi
`,
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
						Annotations: map[string]string{
							"volume.alpha.kubernetes.io/storage-class": "anything",
						},
					},
					Spec: corev1.PersistentVolumeClaimSpec{
						AccessModes: []corev1.PersistentVolumeAccessMode{corev1.ReadWriteOnce},
						Resources: corev1.ResourceRequirements{
							Requests: corev1.ResourceList{corev1.ResourceStorage: resource.MustParse("5Gi")},
						},
					},
				},
			},
		},
	}

	if err := controllerutil.SetControllerReference(instance, statefulSetZero, r.scheme); err != nil {
		return err
	}

	foundS := &appsv1.StatefulSet{}
	err = r.Get(context.TODO(), types.NamespacedName{Name: statefulSetZero.Name, Namespace: statefulSetZero.Namespace}, foundS)
	if err != nil && errors.IsNotFound(err) {
		log.Info("Creating zero statefulsSet", "namespace", statefulSetZero.Namespace, "name", statefulSetZero.Name)
		err = r.Create(context.TODO(), statefulSetZero)
		if err != nil {
			return err
		}
	} else if err != nil {
		return err
	}

	return nil
}
