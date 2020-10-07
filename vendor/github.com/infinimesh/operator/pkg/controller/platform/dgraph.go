package platform

import (
	"context"
	"encoding/base64"
	"fmt"

	"google.golang.org/grpc"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	"github.com/dgraph-io/dgo"
	"github.com/dgraph-io/dgo/protos/api"
	"github.com/go-logr/logr"

	"strings"

	"github.com/infinimesh/infinimesh/pkg/node/dgraph"
	"github.com/infinimesh/infinimesh/pkg/node/nodepb"
	infinimeshv1beta1 "github.com/infinimesh/operator/pkg/apis/infinimesh/v1beta1"
)

const (
	defaultStorage = "10Gi"
)

func setPassword(instance *infinimeshv1beta1.Platform, username, pw string, nodeserverClient nodepb.AccountServiceClient, log logr.Logger) error {
	// Try to login
	_, err := nodeserverClient.Authenticate(context.TODO(), &nodepb.AuthenticateRequest{
		Username: "root",
		Password: pw,
	})
	if err != nil {
		log.Info("Failed to auth with root. Try to create it", "error", err)
	} else {
		log.Info("Logged in with root, password is up to date")
		return nil
	}

	_, err = nodeserverClient.SetPassword(context.TODO(), &nodepb.SetPasswordRequest{
		Username: "root",
		Password: pw,
	})
	if err != nil {
		log.Info("Failed to set pw. Have to create account", "err", err.Error())
	} else {
		log.Info("Set Password to content of secret")
	}

	if err != nil {
		respCreate, err := nodeserverClient.CreateUserAccount(context.TODO(), &nodepb.CreateUserAccountRequest{
			Account: &nodepb.Account{
				Name:    "root",
				IsRoot:  true,
				Enabled: true,
			},
			Password: pw,
		})
		if err != nil {
			log.Error(err, "Failed to create root account")
			return err
		}

		// Write event
		log.Info("Created admin account", "ID", respCreate.Uid)
	}
	return nil
}

func (r *ReconcilePlatform) syncRootPassword(request reconcile.Request, instance *infinimeshv1beta1.Platform) error {
	log := logger.WithName("rootpw")

	hostNodeserver := instance.Name + "-nodeserver." + instance.Namespace + ".svc.cluster.local:8080"
	nodeserverConn, err := grpc.Dial(hostNodeserver, grpc.WithInsecure())
	if err != nil {
		return err
	}

	nodeserverClient := nodepb.NewAccountServiceClient(nodeserverConn)

	randomKey, err := GenerateRandomBytes(32)
	if err != nil {
		return err
	}

	pw := base64.StdEncoding.EncodeToString([]byte(randomKey))

	foundAdminSecret := &corev1.Secret{}
	err = r.Get(context.TODO(), types.NamespacedName{Name: instance.Name + "-root-account", Namespace: instance.Namespace}, foundAdminSecret)
	if err != nil && errors.IsNotFound(err) {
		log.Info("Creating admin secret", "namespace", instance.Namespace, "name", instance.Name+"-root-account")

		secretAdmin := &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      instance.Name + "-root-account",
				Namespace: instance.Namespace,
			},
			StringData: map[string]string{
				"username": "root",
				"password": pw,
			},
		}

		log.Info("gRPC dial OK")
		nodeserverClient := nodepb.NewAccountServiceClient(nodeserverConn)

		err = setPassword(instance, "root", pw, nodeserverClient, log.WithName("setPassword"))
		if err != nil {
			return err
		}

		if err := controllerutil.SetControllerReference(instance, secretAdmin, r.scheme); err != nil {
			return err
		}

		err = r.Create(context.TODO(), secretAdmin)
		if err != nil {
			return err
		}
	} else if err != nil {
		return err
	} else {
		// Exists, sync password in secret to password in dgraph

		secretB64, ok := foundAdminSecret.Data["password"]
		if !ok {
			log.Info("No password field present in secret, ignoring")
			return nil
		}

		secretStr := strings.Trim(string(secretB64), "\n")

		err = setPassword(instance, "root", secretStr, nodeserverClient, log.WithName("setPassword"))
		if err != nil {
			return err
		}
	}
	return nil
}

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

	var pvcSpec corev1.PersistentVolumeClaimSpec
	if instance.Spec.DGraph.Storage == nil {
		pvcSpec = corev1.PersistentVolumeClaimSpec{
			AccessModes: []corev1.PersistentVolumeAccessMode{corev1.ReadWriteOnce},
			Resources: corev1.ResourceRequirements{
				Requests: corev1.ResourceList{corev1.ResourceStorage: resource.MustParse(defaultStorage)},
			},
		}
	} else {
		pvcSpec = *instance.Spec.DGraph.Storage
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
							Image:           "dgraph/dgraph:v1.0.14",
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
					Spec: pvcSpec,
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

	// Alpha
	svcAlpha := &corev1.Service{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Service",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      instance.Name + "-dgraph-alpha",
			Namespace: instance.Namespace,
			Labels: map[string]string{
				"app": instance.Name + "-dgraph-alpha",
			},
		},
		Spec: corev1.ServiceSpec{
			ClusterIP: "None",
			Selector:  map[string]string{"app": instance.Name + "-dgraph-alpha"},
			Ports: []corev1.ServicePort{
				{
					Port:       7080,
					TargetPort: intstr.FromInt(7080),
					Name:       "alpha-grpc-int",
				},
				{
					Port:       9080,
					TargetPort: intstr.FromInt(9080),
					Name:       "alpha-grpc",
				},
			},
		},
	}

	if err := controllerutil.SetControllerReference(instance, svcAlpha, r.scheme); err != nil {
		return err
	}

	foundAlpha := &corev1.Service{}
	err = r.Get(context.TODO(), types.NamespacedName{Name: svcAlpha.Name, Namespace: svcAlpha.Namespace}, foundAlpha)
	if err != nil && errors.IsNotFound(err) {
		log.Info("Creating zero service", "namespace", svcAlpha.Namespace, "name", svcAlpha.Name)
		err = r.Create(context.TODO(), svcAlpha)
		if err != nil {
			return err
		}
	} else if err != nil {
		return err
	}

	// Alpha Statefulset
	statefulSetAlpha := &appsv1.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      instance.Name + "-dgraph-alpha",
			Namespace: instance.Namespace,
		},
		Spec: appsv1.StatefulSetSpec{
			ServiceName: instance.Name + "-dgraph-alpha",
			Replicas:    &replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{"app": instance.Name + "-dgraph-alpha"}, // TODO
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{Labels: map[string]string{"app": instance.Name + "-dgraph-alpha"}},
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
														instance.Name + "-dgraph-alpha",
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
							Name:            "alpha",
							Image:           "dgraph/dgraph:v1.0.14",
							ImagePullPolicy: corev1.PullAlways,
							Ports: []corev1.ContainerPort{
								{
									ContainerPort: 7080,
									Name:          "alpha-grpc-int",
								},
								{
									ContainerPort: 8080,
									Name:          "alpha-http",
								},
								{
									ContainerPort: 9080,
									Name:          "alpha-grpc",
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
dgraph alpha --my=$(hostname -f):7080 --lru_mb 2048 --zero ` + instance.Name + `-dgraph-zero-0.` + instance.Name + `-dgraph-zero.${POD_NAMESPACE}.svc.cluster.local:5080`,
							},
						},
					},
					TerminationGracePeriodSeconds: func() *int64 { val := int64(600); return &val }(),
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
					Spec: pvcSpec,
				},
			},
		},
	}

	if err := controllerutil.SetControllerReference(instance, statefulSetAlpha, r.scheme); err != nil {
		return err
	}

	foundAlphaStatefulset := &appsv1.StatefulSet{}
	err = r.Get(context.TODO(), types.NamespacedName{Name: statefulSetAlpha.Name, Namespace: statefulSetAlpha.Namespace}, foundAlphaStatefulset)
	if err != nil && errors.IsNotFound(err) {
		log.Info("Creating alpha deployment", "namespace", statefulSetAlpha.Namespace, "name", statefulSetAlpha.Name)
		err = r.Create(context.TODO(), statefulSetAlpha)
		if err != nil {
			return err
		}
	} else if err != nil {
		return err
	}

	// TODO: install schema; then update status with that info
	// TODO do this only if necessary
	host := instance.Name + "-dgraph-alpha." + instance.Namespace + ".svc.cluster.local:9080"
	conn, err := grpc.Dial(host, grpc.WithInsecure())
	if err != nil {
		fmt.Println("Failed to connect to dg", err)
	}

	dg := dgo.NewDgraphClient(api.NewDgraphClient(conn))

	err = dgraph.ImportSchema(dg, false)
	if err != nil {
		log.Error(err, "Failed to import schema")
	}
	log.Info("Imported schema")

	err = r.syncRootPassword(request, instance)
	if err != nil {
		log.Error(err, "Failed to sync password")
	}

	return nil
}
