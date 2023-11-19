mkdir -p bin-linux/
rm -r bin-linux/blackwater
env GOOS=linux GOARCH=amd64 go build -o bin-linux/blackwater
echo "compilation done"