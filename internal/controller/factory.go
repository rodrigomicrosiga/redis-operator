package controller

import (
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"

	cachev1alpha1 "github.com/rodrigomicrosiga/redis-operator/api/v1alpha1" // Ajuste para o seu repositório
)

// labelsForRedis retorna os labels base para todos os recursos do Redis
func labelsForRedis(name string) map[string]string {
	return map[string]string{
		"app.kubernetes.io/name":       "redis",
		"app.kubernetes.io/instance":   name,
		"app.kubernetes.io/managed-by": "redis-operator",
	}
}

// buildHeadlessService cria o esqueleto do Serviço de rede interno para os nós do Redis
func buildHeadlessService(cluster *cachev1alpha1.RedisCluster, scheme *runtime.Scheme) (*corev1.Service, error) {
	labels := labelsForRedis(cluster.Name)
	svcName := cluster.Name + "-headless"

	svc := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      svcName,
			Namespace: cluster.Namespace,
			Labels:    labels,
		},
		Spec: corev1.ServiceSpec{
			// O "None" é o que transforma o serviço em Headless
			ClusterIP: corev1.ClusterIPNone,
			Selector:  labels,
			Ports: []corev1.ServicePort{
				{
					Name: "redis",
					Port: 6379,
				},
			},
		},
	}

	// Define o RedisCluster como dono deste Service para o Garbage Collector do K8s
	if err := ctrl.SetControllerReference(cluster, svc, scheme); err != nil {
		return nil, err
	}

	return svc, nil
}

// buildConfigMap cria o mapa de configurações e os scripts de inicialização do Redis
func buildConfigMap(cluster *cachev1alpha1.RedisCluster, scheme *runtime.Scheme) (*corev1.ConfigMap, error) {
	labels := labelsForRedis(cluster.Name)
	cmName := cluster.Name + "-config"

	// O FQDN (Fully Qualified Domain Name) interno do Master dentro do cluster K8s
	masterFQDN := cluster.Name + "-0." + cluster.Name + "-headless." + cluster.Namespace + ".svc.cluster.local"

	// O script verifica se o hostname termina com "-0". Se sim, é o Master. Senão, é Replica.
	setupScript := `#!/bin/sh
echo "Iniciando bootstrap do nó: $HOSTNAME"

if echo "$HOSTNAME" | grep -q "-0$"; then
  echo "=> Assumindo papel de MASTER"
  redis-server /etc/redis/redis.conf
else
  echo "=> Assumindo papel de REPLICA"
  echo "=> Conectando ao Master: ` + masterFQDN + `"
  redis-server /etc/redis/redis.conf --replicaof ` + masterFQDN + ` 6379
fi
`

	cm := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      cmName,
			Namespace: cluster.Namespace,
			Labels:    labels,
		},
		Data: map[string]string{
			"redis.conf": "bind 0.0.0.0\nprotected-mode no\ndir /data\n",
			"setup.sh":   setupScript,
		},
	}

	// Atrela o ConfigMap ao CRD para limpeza automática
	if err := ctrl.SetControllerReference(cluster, cm, scheme); err != nil {
		return nil, err
	}

	return cm, nil
}

// buildStatefulSet cria o controlador dos nós Redis (Master e Replicas)
func buildStatefulSet(cluster *cachev1alpha1.RedisCluster, scheme *runtime.Scheme) (*appsv1.StatefulSet, error) {
	labels := labelsForRedis(cluster.Name)
	stsName := cluster.Name + "-node"
	replicas := cluster.Spec.Size

	// Define a imagem com fallback para a versão padrão
	image := cluster.Spec.RedisImage
	if image == "" {
		image = "redis:7.0-alpine"
	}

	// Permissão de execução para o script bash
	mode := int32(0777)

	sts := &appsv1.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      stsName,
			Namespace: cluster.Namespace,
			Labels:    labels,
		},
		Spec: appsv1.StatefulSetSpec{
			Replicas:    &replicas,
			ServiceName: cluster.Name + "-headless",
			Selector: &metav1.LabelSelector{
				MatchLabels: labels,
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: labels,
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:  "redis",
							Image: image,
							// Subscrevemos o ponto de entrada do Docker para rodar o nosso script inteligente
							Command: []string{"/bin/sh", "/config/setup.sh"},
							Ports: []corev1.ContainerPort{
								{
									Name:          "redis",
									ContainerPort: 6379,
								},
							},
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:      "config-volume",
									MountPath: "/config",
								},
								{
									Name:      "redis-data",
									MountPath: "/data",
								},
							},
						},
					},
					Volumes: []corev1.Volume{
						{
							Name: "config-volume",
							VolumeSource: corev1.VolumeSource{
								ConfigMap: &corev1.ConfigMapVolumeSource{
									LocalObjectReference: corev1.LocalObjectReference{
										Name: cluster.Name + "-config",
									},
									DefaultMode: &mode,
								},
							},
						},
						{
							// Para simplificar o laboratório local, usamos um volume efêmero.
							// Em produção, isso seria substituído por um volumeClaimTemplate (PVC).
							Name: "redis-data",
							VolumeSource: corev1.VolumeSource{
								EmptyDir: &corev1.EmptyDirVolumeSource{},
							},
						},
					},
				},
			},
		},
	}

	if err := ctrl.SetControllerReference(cluster, sts, scheme); err != nil {
		return nil, err
	}

	return sts, nil
}
