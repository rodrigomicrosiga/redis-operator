/*
Copyright 2026.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controller

import (
	"context"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	cachev1alpha1 "github.com/rodrigomicrosiga/redis-operator/api/v1alpha1"
)

// RedisClusterReconciler reconciles a RedisCluster object
type RedisClusterReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=cache.cloud104.io,resources=redisclusters,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=cache.cloud104.io,resources=redisclusters/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=cache.cloud104.io,resources=redisclusters/finalizers,verbs=update
// Permissoes para manipular a infraestrutura base
// +kubebuilder:rbac:groups=apps,resources=statefulsets;deployments,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=core,resources=services;configmaps;pods,verbs=get;list;watch;create;update;patch;delete

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the RedisCluster object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.23.3/pkg/reconcile
func (r *RedisClusterReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := log.FromContext(ctx)

	// 1. Busca a instância do RedisCluster que disparou este evento
	redisCluster := &cachev1alpha1.RedisCluster{}
	err := r.Get(ctx, req.NamespacedName, redisCluster)
	if err != nil {
		if apierrors.IsNotFound(err) {
			log.Info("RedisCluster nao encontrado. Provavelmente foi deletado.")
			return ctrl.Result{}, nil
		}
		log.Error(err, "Falha ao buscar o RedisCluster")
		return ctrl.Result{}, err
	}

	log.Info("Reconciliando RedisCluster", "Name", redisCluster.Name, "Size", redisCluster.Spec.Size)

	// ========================================================================
	// TODO: Chain of Responsibility (A implementar nos próximos passos)
	// ========================================================================
	// Passo 1: Garantir o ConfigMap com os scripts e configurações iniciais
	// ========================================================================
	configMap, err := buildConfigMap(redisCluster, r.Scheme)
	if err != nil {
		log.Error(err, "Falha ao construir o ConfigMap na Factory")
		return ctrl.Result{}, err
	}

	foundCm := &corev1.ConfigMap{}
	err = r.Get(ctx, types.NamespacedName{Name: configMap.Name, Namespace: configMap.Namespace}, foundCm)
	if err != nil && apierrors.IsNotFound(err) {
		log.Info("Criando ConfigMap", "ConfigMap.Namespace", configMap.Namespace, "ConfigMap.Name", configMap.Name)
		err = r.Create(ctx, configMap)
		if err != nil {
			log.Error(err, "Falha ao criar o ConfigMap no cluster")
			return ctrl.Result{}, err
		}
		// Requeue para reavaliar o estado após a criação
		return ctrl.Result{Requeue: true}, nil
	} else if err != nil {
		log.Error(err, "Falha ao buscar o ConfigMap")
		return ctrl.Result{}, err
	}
	// ========================================================================
	// Passo 2: Garantir o Headless Service (Para identidade de rede dos pods)
	// ========================================================================
	headlessSvc, err := buildHeadlessService(redisCluster, r.Scheme)
	if err != nil {
		log.Error(err, "Falha ao construir o Headless Service na Factory")
		return ctrl.Result{}, err
	}

	foundSvc := &corev1.Service{}
	err = r.Get(ctx, types.NamespacedName{Name: headlessSvc.Name, Namespace: headlessSvc.Namespace}, foundSvc)
	if err != nil && apierrors.IsNotFound(err) {
		log.Info("Criando Headless Service", "Service.Namespace", headlessSvc.Namespace, "Service.Name", headlessSvc.Name)
		err = r.Create(ctx, headlessSvc)
		if err != nil {
			log.Error(err, "Falha ao criar o Headless Service no cluster")
			return ctrl.Result{}, err
		}
		// Quando criamos um recurso, retornamos pedindo para o loop rodar de novo (Requeue)
		// para garantir que o próximo passo leia o estado atualizado do cluster.
		return ctrl.Result{Requeue: true}, nil
	} else if err != nil {
		log.Error(err, "Falha ao buscar o Headless Service")
		return ctrl.Result{}, err
	}
	// ========================================================================
	// Passo 3: Garantir o StatefulSet do Redis (Master + Replicas)
	// ========================================================================
	statefulSet, err := buildStatefulSet(redisCluster, r.Scheme)
	if err != nil {
		log.Error(err, "Falha ao construir o StatefulSet na Factory")
		return ctrl.Result{}, err
	}

	foundSts := &appsv1.StatefulSet{}
	err = r.Get(ctx, types.NamespacedName{Name: statefulSet.Name, Namespace: statefulSet.Namespace}, foundSts)
	if err != nil && apierrors.IsNotFound(err) {
		log.Info("Criando StatefulSet", "StatefulSet.Namespace", statefulSet.Namespace, "StatefulSet.Name", statefulSet.Name)
		err = r.Create(ctx, statefulSet)
		if err != nil {
			log.Error(err, "Falha ao criar o StatefulSet no cluster")
			return ctrl.Result{}, err
		}
		// Requeue para que o Kubernetes processe a criação antes de avançarmos
		return ctrl.Result{Requeue: true}, nil
	} else if err != nil {
		log.Error(err, "Falha ao buscar o StatefulSet")
		return ctrl.Result{}, err
	}
	// Passo 4: Garantir o Deployment do Sentinel (O Quórum de HA)
	// Passo 5: Atualizar o Status do CRD

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *RedisClusterReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&cachev1alpha1.RedisCluster{}).
		Named("rediscluster").
		Complete(r)
}
