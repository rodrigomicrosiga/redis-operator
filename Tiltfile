# Tiltfile para Redis Sentinel Operator

# 1. Permite carregar variáveis de ambiente (útil se precisarmos depois)
load('ext://dotenv', 'dotenv')
dotenv()

# 2. Gera os manifestos e o RBAC via Kustomize
yaml = local("kustomize build config/default")
k8s_yaml(yaml)

# 3. Informa ao Tilt como compilar o Operator em um container local
# 'redis-operator:latest' é o nome da imagem que o Kustomize espera no deployment
docker_build(
    'redis-operator:latest',
    context='.',
    dockerfile='Dockerfile',
    # live_update faz o hot-reload do código Go sem precisar recriar o container do zero
    live_update=[
        sync('.', '/workspace'),
        run('cd /workspace && go build -a -o manager main.go', trigger=['./api', './controllers', './internal', 'main.go']),
        restart_container()
    ],
)

# 4. Cria um recurso no painel do Tilt para o CRD de exemplo, facilitando a aplicação
k8s_resource('redis-operator', port_forwards=['8080:8080'])