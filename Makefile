cleanRunner:
	rm -rf bin/micro-ci-runner &
	echo "Cleaned up the 'bin' directory."

cleanServer:
	rm -rf bin/micro-ci-server &
	echo "Cleaned up the 'bin' directory."

cleanTest:
	rm -rf test/out &
	echo "Cleaned up the 'test/out' directory."

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
	mkdir -p test/out
	go test -v ./... -coverprofile=test/out/coverage.out
	go tool cover -html=test/out/coverage.out -o test/out/coverage.html
	echo "Coverage report generated at test/out/coverage.html."
	echo "Launching coverage report..."
	open test/out/coverage.html


buildAndTest: build test

buildAndTestWithCoverage: build coverage
