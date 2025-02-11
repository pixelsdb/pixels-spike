go build -o build/spike-server github.com/AgentGuo/spike/cmd/server

# dlv exec ./spike-server --headless --listen=:5006 --api-version=2 --accept-multiclient -- -f spike.yaml