cleanRunner:
	rm -rf bin/micro-ci-runner &
	echo "Cleaned up the 'bin' directory."

cleanServer:
	rm -rf bin/micro-ci-server &
	echo "Cleaned up the 'bin' directory."

clean: cleanRunner cleanServer
	echo "Cleaned up all build artifacts for Micro CI server and runner."

buildRunner: cleanRunner
	echo "Starting the build process for Micro CI runner..."
	go build -o bin/micro-ci-runner ./cmd/micro-ci-runner/main.go

buildServer: cleanServer
	echo "Starting the build process for Micro CI server..."
	go build -o bin/micro-ci-server ./cmd/micro-ci-server/main.go

build: clean buildRunner buildServer
	echo "Build process for Micro CI server and runner completed."

runServer: buildServer
	@echo "Starting Micro CI Server..."
	./bin/micro-ci-server
	echo "Micro CI Server is now running."

runRunner: buildRunner
	@echo "Starting Micro CI Runner..."
	./bin/micro-ci-runner
	echo "Micro CI Runner is now running."

buildAndRunServer: buildServer runServer

buildAndRunRunner: buildRunner runRunner