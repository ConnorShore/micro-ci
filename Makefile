cleanRunner:
	rm -rf bin/micro-ci-runner &
	echo "Cleaned up the 'bin' directory."

cleanServer:
	rm -rf bin/micro-ci-server &
	echo "Cleaned up the 'bin' directory."

cleanTest:
	rm -rf bin/test &
	echo "Cleaned up the 'bin/test' directory."

clean: cleanRunner cleanServer cleanTest
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

test:
	@echo "Running tests with verbose output..."
	go test -v ./...
	echo "All tests completed."

coverage:
	@echo "Running tests with coverage report..."
	mkdir -p bin/test
	go test -v ./... -coverprofile=bin/test/coverage.out
	go tool cover -html=bin/test/coverage.out -o bin/test/coverage.html
	echo "Coverage report generated at bin/test/coverage.html."
	echo "Launching coverage report..."
	open bin/test/coverage.html


buildAndTest: build test

buildAndTestWithCoverage: build coverage
