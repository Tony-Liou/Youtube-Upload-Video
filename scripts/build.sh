# This script builds the application from source

# Change into the root level directory of this app
cd ..

# Delete old dir
echo "Removing old directory..."
rm -f bin/*
mkdir -p bin/

# Instruct gox to build statically linked binaries
export CGO_ENABLED=0

echo "Start building..."
go build HttpServer.go streaminghandler.go -o bin/HttpServer
echo "Finshed"