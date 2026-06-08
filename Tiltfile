# Tiltfile para Redis Sentinel Operator

# 1. Permite carregar variáveis de ambiente (útil se precisarmos depois)
load('ext://dotenv', 'dotenv')
dotenv()

# 2. Gera os manifestos e o RBAC via Kustomize
yaml = local("kustomize build config/default")
k8s_yaml(yaml)

# 3. Informa ao Tilt como compilar o Operator em um container local
docker_build(
    'controller',
    context='.',
    dockerfile='Dockerfile'
)

# 4. Cria um recurso no painel do Tilt para o Operator
k8s_resource('redis-operator-controller-manager', port_forwards=['8080:8080'])