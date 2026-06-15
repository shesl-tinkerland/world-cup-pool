param(
    [ValidateRange(1024, 65535)]
    [int]$Port = 8091
)

$ErrorActionPreference = 'Stop'

if ($Port -eq 8090) {
    throw 'Refusing to use port 8090. That port is reserved for production in this workspace.'
}

$prodStatus = docker ps --filter 'name=^/fhun_tips$' --format '{{.Names}} {{.Status}}' 2>$null
if ($prodStatus) {
    Write-Host "Production container left untouched: $prodStatus"
}

$alternateTest = docker ps -a --filter 'name=^/world_cup_pool_test$' --format '{{.Names}}' 2>$null
if ($alternateTest) {
    docker rm -f world_cup_pool_test 2>$null
}

$env:TEST_HTTP_PORT = [string]$Port
$env:TEST_WMP_DEV = '1'
$env:TEST_RESULTS_SOURCE = 'openfootball'

docker compose -f docker-compose.test.yml -p fhun_tips_test up --build -d

Write-Host "Test app: http://localhost:$Port"
Write-Host 'Container: fhun_tips_test'
Write-Host 'Data volume: fhun_tips_test_pb_data_test'