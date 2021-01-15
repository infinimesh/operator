package platform

import (
	"context"

	infinimeshv1beta1 "github.com/infinimesh/operator/pkg/apis/infinimesh/v1beta1"
	batchv1 "k8s.io/api/batch/v1"
	v1beta1 "k8s.io/api/batch/v1beta1"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

func (r *ReconcilePlatform) reconcileHardDeleteNamespace(request reconcile.Request, instance *infinimeshv1beta1.Platform) error {
	log := logger.WithName("HardDeleteNamespace")
	cronjob := &v1beta1.CronJob{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "demo-cronjob",
			Namespace: "default",
		},
		Spec: v1beta1.CronJobSpec{
			Schedule:          "*/1 * * * *",
			ConcurrencyPolicy: v1beta1.ForbidConcurrent,
			JobTemplate: v1beta1.JobTemplateSpec{
				Spec: batchv1.JobSpec{
					Template: v1.PodTemplateSpec{
						Spec: v1.PodSpec{
							RestartPolicy: corev1.RestartPolicyOnFailure,
							Containers: []corev1.Container{
								{
									Name:            "hello",
									Image:           "busybox",
									ImagePullPolicy: corev1.PullAlways,
									Command: []string{
										"/bin/sh", "-c", "date; echo Hello from the Kubernetes cluster",
									},
								},
							},
						},
					},
				},
			},
		},
	}
	if err := controllerutil.SetControllerReference(instance, cronjob, r.scheme); err != nil {
		return err
	}

	foundS := &v1beta1.CronJob{}
	err := r.Get(context.TODO(), types.NamespacedName{Name: cronjob.Name, Namespace: cronjob.Namespace}, foundS)
	if err != nil && errors.IsNotFound(err) {
		log.Info("Creating demo cronjob", "namespace", cronjob.Namespace, "name", cronjob.Name)
		err = r.Create(context.TODO(), cronjob)
		if err != nil {
			return err
		}
	} else if err != nil {
		return err
	}
	return nil

}
