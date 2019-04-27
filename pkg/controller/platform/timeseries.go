package platform

import (
	"context"
	"reflect"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	infinimeshv1beta1 "github.com/infinimesh/operator/pkg/apis/infinimesh/v1beta1"
)

func (r *ReconcilePlatform) reconcileTimeseries(request reconcile.Request, instance *infinimeshv1beta1.Platform) error {
	log := logger.WithName("time series")
	{
		deploymentName := instance.Name + "-timescale-connector"
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
						"storage": "10Gi",
					},
				},
			},
			"terminationPolicy": "DoNotTerminate",
		},
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

	// TODO updating not implemented

	return nil
}
