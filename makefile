# build main applications (server + agent)
BUILD_APPS:
	cd cmd/server
	go build
	cd ../agent
	go build
	cd ../..

# build additional linter
BUILD_LINTER:
	cd cmd/linter
	go build
	cd ../..

# build reset tool for code generation
BUILD_RESET:
	cd cmd/reset
	go build
	cd ../..

# additional tools for tests setup
BUILD_TEST_TOOLS:
	cd cmd/tools/spammer
	go build
	cd ../../..

BUILD_ALL:
	make BUILD_APPS
	make BUILD_LINTER
	make BUILD_RESET
	make BUILD_TEST_TOOLS

