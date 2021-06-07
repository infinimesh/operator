package platform

import (
	"context"

	infinimeshv1beta1 "github.com/infinimesh/operator/pkg/apis/infinimesh/v1beta1"
	batchv1 "k8s.io/api/batch/v1"
	v1beta1 "k8s.io/api/batch/v1beta1"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/api/core/v1"

	v1beta1rbac "k8s.io/api/rbac/v1beta1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

func (r *ReconcilePlatform) reconcileResetRootAccountPwd(request reconcile.Request, instance *infinimeshv1beta1.Platform) error {
	log := logger.WithName("Reset root account pwd")
	role := &v1beta1rbac.Role{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Role",
			APIVersion: "rbac.authorization.k8s.io",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "reset-pwd",
			Namespace: "default",
		},
		Rules: []v1beta1rbac.PolicyRule{
			v1beta1rbac.PolicyRule{
				APIGroups: []string{""},
				Resources: []string{"secrets"},
				Verbs:     []string{"delete", "get"},
			},
		},
	}
	roleBinding := &v1beta1rbac.RoleBinding{
		TypeMeta: metav1.TypeMeta{
			Kind:       "RoleBinding",
			APIVersion: "rbac.authorization.k8s.io/v1beta1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "reset-pwd",
			Namespace: "default",
		},
		Subjects: []v1beta1rbac.Subject{
			v1beta1rbac.Subject{
				Kind:      "ServiceAccount",
				Name:      "reset-root-account-pwd",
				Namespace: "default",
			},
		},
		RoleRef: v1beta1rbac.RoleRef{
			APIGroup: "",
			Kind:     "Role",
			Name:     "reset-pwd",
		},
	}

	svc_account := &v1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "reset-root-account-pwd",
			Namespace: "default",
		},
	}
	cronjob := &v1beta1.CronJob{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "delete-root-account-secret",
			Namespace: "default",
		},
		Spec: v1beta1.CronJobSpec{
			Schedule:          "0 0 * * *",
			ConcurrencyPolicy: v1beta1.ForbidConcurrent,
			JobTemplate: v1beta1.JobTemplateSpec{
				Spec: batchv1.JobSpec{
					Template: v1.PodTemplateSpec{
						Spec: v1.PodSpec{
							RestartPolicy:      corev1.RestartPolicyOnFailure,
							ServiceAccountName: "reset-root-account-pwd",
							Containers: []corev1.Container{
								{
									Name:            "kubectl",
									Image:           "garland/kubectl:1.10.4",
									ImagePullPolicy: corev1.PullAlways,
									Command: []string{
										"/bin/sh", "-c", "kubectl delete secret " + instance.Name + "-root-account -n default;",
									},
								},
							},
						},
					},
				},
			},
		},
	}
	//------Creating Role-------//
	if err := controllerutil.SetControllerReference(instance, role, r.scheme); err != nil {
		return err
	}

	found_role := &v1beta1rbac.Role{}
	err := r.Get(context.TODO(), types.NamespacedName{Name: role.Name, Namespace: role.Namespace}, found_role)
	if err != nil && errors.IsNotFound(err) {
		log.Info("Creating role for root account password", "namespace", role.Namespace, "name", role.Name)
		err = r.Create(context.TODO(), role)
		if err != nil {
			return err
		}
	} else if err != nil {
		return err
	}
	//------Creating Role Binding-------//
	if err := controllerutil.SetControllerReference(instance, roleBinding, r.scheme); err != nil {
		return err
	}

	found_roleBinding := &v1beta1rbac.RoleBinding{}
	err = r.Get(context.TODO(), types.NamespacedName{Name: roleBinding.Name, Namespace: roleBinding.Namespace}, found_roleBinding)
	if err != nil && errors.IsNotFound(err) {
		log.Info("Creating role binding for root account password", "namespace", roleBinding.Namespace, "name", roleBinding.Name)
		err = r.Create(context.TODO(), roleBinding)
		if err != nil {
			return err
		}
	} else if err != nil {
		return err
	}
	//------Creating Service Account-------//
	if err = controllerutil.SetControllerReference(instance, svc_account, r.scheme); err != nil {
		return err
	}

	found_svc_account := &v1.ServiceAccount{}
	err = r.Get(context.TODO(), types.NamespacedName{Name: svc_account.Name, Namespace: svc_account.Namespace}, found_svc_account)
	if err != nil && errors.IsNotFound(err) {
		log.Info("Creating service account root account password", "namespace", svc_account.Namespace, "name", svc_account.Name)
		err = r.Create(context.TODO(), svc_account)
		if err != nil {
			return err
		}
	} else if err != nil {
		return err
	}
	//------Creating Cron Job-------//
	if err := controllerutil.SetControllerReference(instance, cronjob, r.scheme); err != nil {
		return err
	}
	foundS := &v1beta1.CronJob{}
	err = r.Get(context.TODO(), types.NamespacedName{Name: cronjob.Name, Namespace: cronjob.Namespace}, foundS)
	if err != nil && errors.IsNotFound(err) {
		log.Info("Creating cronjob for resetting root account password", "namespace", cronjob.Namespace, "name", cronjob.Name)
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
