build: clean
	echo "Starting the build process for Micro CI server and runner..."
	go build -o bin/micro-ci-server ./cmd/micro-ci-server/main.go
	go build -o bin/micro-ci-runner ./cmd/micro-ci-runner/main.go
	echo "Build completed successfully. Binaries are located in the 'bin' directory."

runServer:
	./bin/micro-ci-server
	echo "Micro CI Server is now running."

runRunner:
	./bin/micro-ci-runner
	echo "Micro CI Runner is now running."

clean:
	rm -rf bin/* &
	echo "Cleaned up the 'bin' directory."