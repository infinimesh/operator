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
			Name:      "harddeletenamespace",
			Namespace: "default",
		},
		Spec: v1beta1.CronJobSpec{
			Schedule:          "0/1 0 * * *",
			ConcurrencyPolicy: v1beta1.ForbidConcurrent,
			JobTemplate: v1beta1.JobTemplateSpec{
				Spec: batchv1.JobSpec{
					Template: v1.PodTemplateSpec{
						Spec: v1.PodSpec{
							RestartPolicy: corev1.RestartPolicyOnFailure,
							Containers: []corev1.Container{
								{
									Name:            "harddelete",
									Image:           "curlimages/curl",
									ImagePullPolicy: corev1.PullAlways,
									EnvFrom: []corev1.EnvFromSource{
										{
											SecretRef: &corev1.SecretEnvSource{
												LocalObjectReference: corev1.LocalObjectReference{
													Name: instance.Name + "-root-account",
												},
											},
										},
									},
									Command: []string{
										"/bin/sh", "-c", "echo START;",
										"echo START;", "printenv;",
										"echo $MY_INFINIMESH_APISERVER_REST_SERVICE_HOST;",
										"temptoken=`(curl --location --request POST 'https://'\"$MY_INFINIMESH_APISERVER_REST_SERVICE_HOST\"'/account/token' --header 'Content-Type:application/json' --data-raw '{\"password\":\"'\"$password\"'\",\"username\":\"'\"$username\"'\"}' | sed -n '/ *\"token\":*\"/ { s///; s/\".*//; p; }')`;",
										"token=`echo \"${temptoken:1}\"`;",
										"echo $token;",
										"curl -X DELETE https://api.infinimesh.dev/namespaces/0xeab0/true -H 'Authorization:bearer '\"$token\"'';",
										"echo END;",
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
		log.Info("Creating cronjob for hard delete namespace", "namespace", cronjob.Namespace, "name", cronjob.Name)
		err = r.Create(context.TODO(), cronjob)
		if err != nil {
			return err
		}
	} else if err != nil {
		return err
	}
	return nil
	//dummy commit to build
}
