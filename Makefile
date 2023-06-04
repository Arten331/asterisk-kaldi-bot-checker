SHELL := '/bin/bash'
ROOT_DIR:=$(shell dirname $(realpath $(firstword $(MAKEFILE_LIST))))


push_test_image:
	cd .k8s/docker/test
	docker build -t {{you_registry_go_test_container_link}}:latest -f ./Dockerfile-tests .
	docker image push {{you_registry_go_test_container_link}}:latest

lint:
	docker run --rm -v $(ROOT_DIR):/app -w /app registry.gitlab.com/gitlab-org/gitlab-build-images:golangci-lint-alpine /bin/bash -c \
      "golangci-lint run --issues-exit-code 0 --out-format code-climate | tee gl-code-quality-report.json | jq -r '.[] | \"\(.location.path):\(.location.lines.begin) \(.description)\"' \
       && golint_warnings=`jq -r  length gl-code-quality-report.json` > .golint_warnings && cat .golint_warnings "

lint_code_climate:
		[ -e .golangci.yml ] || cp /golangci/.golangci.yml $(ROOT_DIR)
		docker run \
    	  --interactive --tty --rm \
    	  --env CODECLIMATE_CODE="$(ROOT_DIR)" \
    	  --volume "$(ROOT_DIR)":/code \
    	  --volume /var/run/docker.sock:/var/run/docker.sock \
    	  --volume /tmp/cc:/tmp/cc \
    	  codeclimate/codeclimate analyze -f json -e golint | tee gl-code-quality-report.json
#    	lint warnings count
		lint_warnings=$$(jq -r length gl-code-quality-report.json) && echo "golangcilint warnings: $${lint_warnings}"; \
		shield_lint_color="red"; if [[ $$lint_warnings -lt 10 ]]; then shield_lint_color="green"; fi; \
		shield_lint_link=https://img.shields.io/badge/issues-$${lint_warnings}-$${shield_lint_color}; \
		curl -o golint_issues.svg $${shield_lint_link}; \

code_climate_help:
	docker run \
	  --interactive --tty --rm \
	  --env CODECLIMATE_CODE="$PWD" \
	  --volume "$(ROOT_DIR)":/code \
	  --volume /var/run/docker.sock:/var/run/docker.sock \
	  --volume /tmp/cc:/tmp/cc \
	  codeclimate/codeclimate help

tests:
	docker run -v $(ROOT_DIR):/app -w /app {{you_registry_go_test_container_link}} /bin/bash -c "gotestsum --junitfile report.xml --format testname -- --tags 'test' -covermode=count -coverprofile .coverage.txt ./..."

tests_integration:
	docker run -v $(ROOT_DIR):/app -w /app {{you_registry_go_test_container_link}} /bin/bash -c "gotestsum --junitfile report.integration.xml --format testname -- --tags 'test integration' -covermode=count -coverprofile .coverage.txt ./..."
