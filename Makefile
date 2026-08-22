.PHONY: help test test-all test-quality test-race test-scaling test-compat test-fuzz test-bench

help:
	@echo 'Testing targets:'
	@echo '  test          fast Go unit and integration suite'
	@echo '  test-quality  formatting, vet, lint, and diff checks'
	@echo '  test-race     full Go suite with the race detector'
	@echo '  test-scaling  parser/evaluator growth invariants (3 runs)'
	@echo '  test-compat   complete browser compatibility oracle'
	@echo '  test-fuzz     unified XPath parser fuzz target'
	@echo '  test-bench    benchmarks with allocation reporting'
	@echo '  test-all      sequential pre-merge verification'

# Fast default for normal edit/test loops.
test:
	@./scripts/test/go.sh unit

test-quality:
	@./scripts/test/quality.sh

test-race:
	@./scripts/test/go.sh race

test-scaling:
	@./scripts/test/go.sh scaling

test-compat:
	@./scripts/test/compat.sh

test-fuzz:
	@./scripts/test/go.sh fuzz

test-bench:
	@./scripts/test/go.sh bench

# Complete pre-merge verification. Keep the layers sequential even when the
# caller enables parallel make; timing assertions must not compete with race.
# Fuzzing and benchmarks remain explicit because their duration is controlled
# by the caller and machine.
test-all:
	@$(MAKE) test-quality
	@$(MAKE) test
	@$(MAKE) test-race
	@$(MAKE) test-scaling
	@$(MAKE) test-compat
