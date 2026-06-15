$ErrorActionPreference = 'Stop'

docker compose -f docker-compose.test.yml -p fhun_tips_test down
docker rm -f world_cup_pool_test 2>$null

Write-Host 'Stopped only the isolated test app (fhun_tips_test).'