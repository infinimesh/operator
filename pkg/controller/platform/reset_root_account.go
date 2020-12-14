package platform

import (
	"context"
	"fmt"

	infinimeshv1beta1 "github.com/infinimesh/operator/pkg/apis/infinimesh/v1beta1"
	v1beta1 "k8s.io/api/batch/v1beta1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

func (r *ReconcilePlatform) reconcileResetRootAccount(request reconcile.Request, instance *infinimeshv1beta1.Platform) error {

	log := logger.WithName("Reset Root Account Pwd")
	job := &v1beta1.CronJob{
		ObjectMeta: metav1.ObjectMeta{
			Name: "example",
		},
		Spec: v1beta1.CronJobSpec{
			Schedule:          "* * * * *",
			ConcurrencyPolicy: v1beta1.ForbidConcurrent,
			JobTemplate: v1beta1.JobTemplate{
				spec: v1beta1.JobTemplateSpec{
					Template: corev1.PodTemplateSpec{
						Spec: corev1.PodSpec{
							RestartPolicy: "Never",
							Containers: []corev1.Container{
								{
									Name:  "cli",
									Image: "busybox",
									Command: []string{
										"/bin/bash",
										"-c",
										"echo 1",
									},
									ImagePullPolicy: "Always",
								},
							},
						},
					},
				},
			},
		},
	}

	// config := &rest.Config{
	// 	Host: "http://localhost:8080",
	// }

	// clientset, err := kubernetes.NewForConfig(config)
	// if err != nil {
	// 	panic(err)
	// }

	// _, err = clientset.Batchv1beta1().CronJobs("default").Create(job)
	// if err != nil {
	// 	panic(err)
	// }

	fmt.Println("Deployed example cronjob")

	if err := controllerutil.SetControllerReference(instance, job, r.scheme); err != nil {
		return err
	}

	found := &v1beta1.CronJob{}
	err := r.Get(context.TODO(), types.NamespacedName{Name: job.Name, Namespace: job.Namespace}, found)
	if err != nil && errors.IsNotFound(err) {
		log.Info("Deployed example cronjob", "namespace", job.Namespace, "name", job.Name)
		err = r.Create(context.TODO(), job)
		if err != nil {
			return err
		}
	}
	return nil
}
